package ui

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ethereum/go-ethereum/common"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/storage"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/ui/assets"
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

	spinner spinner.Model
	active  tab
	width   int
	height  int

	// Pestaña Cuentas
	input     textinput.Model
	addErr    error
	accCursor int

	// Pestaña Balances
	balState   loadState
	balResults []chain.BalanceResult
	balCursor  int

	// Pestaña Transacciones
	txChainID uint64
	txState   loadState
	txs       []txRow
	txErr     error
	txCursor  int
	txWallet  common.Address
}

// NewModel construye el modelo raíz inyectando todas las dependencias: cliente
// de cadena, almacenamiento de wallets, redes efectivas (con overrides de config)
// e intervalo de refresco.
func NewModel(client *chain.Client, wallets *storage.Wallets, networks []chain.Network, refresh time.Duration, txProvider chain.TxProvider) Model {
	styles := DefaultStyles()
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(styles.Spinner),
	)

	ti := textinput.New()
	ti.Placeholder = "0x… address EVM, luego Enter"
	ti.Prompt = "› "
	// Holgura sobre los 42 chars de una address para que un pegado con espacios
	// alrededor no se trunque; el TrimSpace + validación al añadir lo sanean.
	ti.CharLimit = 64
	ti.Focus() // arrancamos en la pestaña Cuentas, con el input listo

	return Model{
		styles:     styles,
		client:     client,
		wallets:    wallets,
		networks:   networks,
		refresh:    refresh,
		txProvider: txProvider,
		spinner:    sp,
		input:      ti,
		active:     tabAccounts,
		// El historial de txs en la v1 se consulta sobre Ethereum mainnet.
		txChainID: chain.ChainEthereum,
	}
}

// Init arranca el bucle de refresco periódico de balances.
func (m Model) Init() tea.Cmd {
	return refreshTickCmd(m.refresh)
}

// Update enruta cada mensaje. Las teclas globales (salir, cambiar pestaña) se
// manejan aquí; el resto se delega a la pestaña activa.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if msg.Width > 20 {
			m.input.SetWidth(msg.Width - 12)
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
		// 'q' sale salvo en Cuentas, donde se está escribiendo en el input.
		if msg.String() == "q" && m.active != tabAccounts {
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
		if m.balState != stateLoading {
			return m, nil // dejamos morir el spinner al terminar la carga
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

// View renderiza título + pestañas + cuerpo + ayuda contextual.
func (m Model) View() tea.View {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render(assets.Title))
	b.WriteString("\n\n")
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")
	b.WriteString(m.renderBody())
	b.WriteString("\n\n")
	b.WriteString(m.styles.Faint.Render(m.helpLine()))

	return tea.NewView(b.String())
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

func (m Model) renderBody() string {
	var content string
	switch m.active {
	case tabAccounts:
		content = m.renderAccounts()
	case tabBalances:
		content = m.renderBalances()
	case tabTransactions:
		content = m.renderTransactions()
	}

	panel := m.styles.Panel
	if m.width > 0 {
		panel = panel.Width(m.width - 6)
	}
	return panel.Render(content)
}

// helpLine devuelve la línea de ayuda según la pestaña activa.
func (m Model) helpLine() string {
	switch m.active {
	case tabAccounts:
		return "tab pestaña · enter añadir · ctrl+d borrar · ↑↓ seleccionar · ctrl+c salir"
	case tabBalances:
		return "tab pestaña · ↑↓ navegar · r recargar · q salir"
	case tabTransactions:
		return "tab pestaña · ↑↓ navegar · r recargar · q salir"
	default:
		return "tab pestaña · q salir"
	}
}
