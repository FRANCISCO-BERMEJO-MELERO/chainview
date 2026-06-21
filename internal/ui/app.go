package ui

import (
	"fmt"
	"math/big"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ethereum/go-ethereum/common"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/storage"
)

// loadState modela el ciclo de vida de la carga de balances.
type loadState int

const (
	stateIdle loadState = iota
	stateLoading
	stateLoaded
	stateError
)

// tab identifica cada pestaña navegable de la TUI.
type tab int

const (
	tabAccounts tab = iota
	tabBalances
	tabTransactions
)

var orderedTabs = []tab{tabAccounts, tabBalances, tabTransactions}

func (t tab) title() string {
	switch t {
	case tabAccounts:
		return "Cuentas"
	case tabBalances:
		return "Balances"
	case tabTransactions:
		return "Transacciones"
	default:
		return ""
	}
}

// Model es el modelo raíz de la TUI (patrón Elm).
type Model struct {
	styles     Styles
	client     *chain.Client
	wallets    *storage.Wallets
	networks   []chain.Network
	refresh    time.Duration
	txProvider chain.TxProvider
	ens        *chain.ENSResolver

	// ensNames cachea en el Model los nombres ENS ya resueltos (address -> nombre)
	// para mostrarlos sin tocar la red en el render.
	ensNames map[common.Address]string

	// gas guarda el último gas price por red (en wei) y gasPrev el anterior, para
	// calcular la tendencia ↑/↓ en el header.
	gas     map[uint64]*big.Int
	gasPrev map[uint64]*big.Int

	spinner spinner.Model
	active  tab
	width   int
	height  int
	// contentW/contentH son el área útil interior del frame, recalculada en cada
	// WindowSizeMsg para dimensionar el contenido y los modales.
	contentW int
	contentH int

	// Pestaña Cuentas
	input         textinput.Model
	addErr        error
	accCursor     int
	resolvingName string // nombre ENS que se está resolviendo para añadir, si hay

	// Pestaña Balances
	balState   loadState
	balResults []chain.BalanceResult
	balCursor  int

	// Pestaña Transacciones
	txChainID    uint64
	txState      loadState
	txs          []txRow
	txErr        error
	txCursor     int
	txWallet     common.Address
	txDetailOpen bool           // modal de detalle de la tx seleccionada
	txViewport   viewport.Model // contenido scrollable del modal
}

// NewModel construye el modelo raíz inyectando todas las dependencias: cliente
// de cadena, almacenamiento de wallets, redes efectivas (con overrides de config)
// e intervalo de refresco.
func NewModel(client *chain.Client, wallets *storage.Wallets, networks []chain.Network, refresh time.Duration, txProvider chain.TxProvider, ens *chain.ENSResolver) Model {
	styles := DefaultStyles()
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(styles.Spinner),
	)

	ti := textinput.New()
	ti.Placeholder = "0x… address o nombre ENS (vitalik.eth), luego Enter"
	ti.Prompt = "› "
	// Holgura sobre los 42 chars de una address para que un pegado con espacios
	// alrededor no se trunque; el TrimSpace + validación al añadir lo sanean.
	ti.CharLimit = 64
	ti.Focus() // arrancamos en la pestaña Cuentas, con el input listo

	// Viewport del modal de detalle de tx. Tamaño por defecto razonable hasta que
	// llegue el primer WindowSizeMsg.
	vp := viewport.New()
	vp.SetWidth(72)
	vp.SetHeight(16)

	return Model{
		styles:     styles,
		client:     client,
		wallets:    wallets,
		networks:   networks,
		refresh:    refresh,
		txProvider: txProvider,
		ens:        ens,
		ensNames:   make(map[common.Address]string),
		gas:        make(map[uint64]*big.Int),
		gasPrev:    make(map[uint64]*big.Int),
		spinner:    sp,
		input:      ti,
		txViewport: vp,
		active:     tabAccounts,
		// El historial de txs en la v1 se consulta sobre Ethereum mainnet.
		txChainID: chain.ChainEthereum,
	}
}

// Init arranca los bucles de refresco (balances y gas), una primera lectura de
// gas inmediata y la resolución ENS de las wallets ya seguidas.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		refreshTickCmd(m.refresh),
		gasTickCmd(m.refresh),
		m.fetchGasCmd(),
		m.resolveWalletNamesCmd(),
	)
}

// Update enruta cada mensaje. Las teclas globales (salir, cambiar pestaña) se
// manejan aquí; el resto se delega a la pestaña activa.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.contentW, m.contentH = contentDims(msg.Width, msg.Height)
		if m.contentW > 8 {
			m.input.SetWidth(m.contentW - 4)
		}
		// El modal de detalle de tx vive en el área de contenido; le dejamos 2
		// líneas para su propia pista de ayuda.
		m.txViewport.SetWidth(m.contentW)
		if m.contentH > 2 {
			m.txViewport.SetHeight(m.contentH - 2)
		}

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.active = m.nextTab(1)
			return m, m.onEnterTab()
		case "shift+tab":
			m.active = m.nextTab(-1)
			return m, m.onEnterTab()
		}
		// 'q' sale salvo en Cuentas (se está escribiendo en el input) o con el
		// modal de detalle abierto (ahí 'q' no debe cerrar la app).
		if msg.String() == "q" && m.active != tabAccounts && !m.txDetailOpen {
			return m, tea.Quit
		}
		switch m.active {
		case tabAccounts:
			return m.updateAccounts(msg)
		case tabBalances:
			return m.updateBalances(msg)
		case tabTransactions:
			return m.updateTransactions(msg)
		}
		return m, nil

	case balancesMsg:
		m.balResults = msg.results
		m.balState = stateLoaded
		m.clampBalCursor()
		return m, nil

	case txsLoadedMsg:
		// Descartamos resultados de una wallet que ya no es la seleccionada.
		if msg.wallet == m.txWallet {
			m.txs = msg.rows
			m.txErr = nil
			m.txState = stateLoaded
			m.clampTxCursor()
		}
		return m, nil

	case txsErrMsg:
		if msg.wallet == m.txWallet {
			m.txErr = msg.err
			m.txState = stateError
		}
		return m, nil

	case ensResolvedMsg:
		for addr, name := range msg.names {
			m.ensNames[addr] = name
		}
		return m, nil

	case ensAddMsg:
		m.resolvingName = ""
		if !msg.ok {
			m.addErr = fmt.Errorf("no se pudo resolver %q en ENS", msg.name)
			return m, nil
		}
		if err := m.wallets.Add(msg.addr.Hex()); err != nil {
			m.addErr = err
			return m, nil
		}
		m.addErr = nil
		m.ensNames[msg.addr] = msg.name // ya conocemos el nombre, lo cacheamos
		m.input.Reset()
		m.balState = stateIdle
		m.clampAccCursor()
		return m, nil

	case gasTickMsg:
		// El gas se refresca siempre (el header está visible en todas las tabs).
		return m, tea.Batch(gasTickCmd(m.refresh), m.fetchGasCmd())

	case gasMsg:
		for _, r := range msg.results {
			if r.Err != nil || r.Wei == nil {
				continue // conservamos el último valor bueno de esa red
			}
			if cur, ok := m.gas[r.ChainID]; ok {
				m.gasPrev[r.ChainID] = cur // el actual pasa a ser el anterior
			}
			m.gas[r.ChainID] = r.Wei
		}
		return m, nil

	case refreshTickMsg:
		// El tick siempre se reprograma; solo refrescamos si estamos viendo
		// Balances, hay wallets y no hay ya una carga en vuelo (anti-solape).
		cmds := []tea.Cmd{refreshTickCmd(m.refresh)}
		if m.active == tabBalances && m.balState != stateLoading && m.wallets.Len() > 0 {
			m.balState = stateLoading
			cmds = append(cmds, m.spinner.Tick, m.fetchBalancesCmd())
		}
		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		// Mantenemos el spinner vivo mientras haya cualquier carga en curso
		// (balances o transacciones); si no, lo dejamos morir.
		if m.balState != stateLoading && m.txState != stateLoading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	default:
		// Mensajes que no manejamos arriba (pegado con bracketed paste vía
		// tea.PasteMsg, parpadeo del cursor, etc.) se reenvían al textinput
		// cuando estamos en Cuentas, que es quien sabe qué hacer con ellos.
		if m.active == tabAccounts {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

// onEnterTab ajusta el foco del input y lanza la carga de balances al entrar en
// la pestaña Balances por primera vez.
func (m *Model) onEnterTab() tea.Cmd {
	m.txDetailOpen = false // cambiar de pestaña cierra el modal de detalle
	switch m.active {
	case tabAccounts:
		return m.input.Focus()
	case tabBalances:
		m.input.Blur()
		if m.balState == stateIdle && m.wallets.Len() > 0 {
			m.balState = stateLoading
			return tea.Batch(m.spinner.Tick, m.fetchBalancesCmd())
		}
	case tabTransactions:
		m.input.Blur()
		return m.loadTxsCmd()
	default:
		m.input.Blur()
	}
	return nil
}

func (m Model) nextTab(delta int) tab {
	n := len(orderedTabs)
	return tab((int(m.active) + delta + n) % n)
}

// View dibuja toda la app como un frame que llena el terminal (ver layout.go).
func (m Model) View() tea.View {
	return tea.NewView(m.renderFrame())
}

func (m Model) renderTabs() string {
	rendered := make([]string, len(orderedTabs))
	for i, t := range orderedTabs {
		if t == m.active {
			rendered[i] = m.styles.TabActive.Render(t.title())
		} else {
			rendered[i] = m.styles.TabInactive.Render(t.title())
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, rendered...)
}
