package ui

import (
	"context"
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

	// Valoración fiat (1.1): proveedor de precios, moneda configurada y el último
	// precio por activo (nativo o token) tasado, para mostrar valor por celda y el
	// total de la cartera.
	priceProvider chain.PriceProvider
	fiatCurrency  string
	prices        map[chain.PriceQuery]float64

	// Descubrimiento de tokens ERC-20 (1.2). Keyless (Blockscout); nil lo desactiva.
	tokenProvider chain.TokenBalanceProvider

	// Tema (2.1): themePref es la preferencia (dark/light/auto); themeNow es la
	// paleta efectiva ya aplicada (dark/light). Con "auto" se resuelve al recibir
	// el color de fondo del terminal (tea.BackgroundColorMsg).
	themePref string
	themeNow  string

	// ensNames cachea en el Model los nombres ENS ya resueltos (address -> nombre)
	// para mostrarlos sin tocar la red en el render.
	ensNames map[common.Address]string

	// gas guarda el último gas price por red (en wei) y gasPrev el anterior, para
	// calcular la tendencia ↑/↓ en el header.
	gas     map[uint64]*big.Int
	gasPrev map[uint64]*big.Int

	// Estado vivo para la barra inferior.
	lastGas  time.Time // última lectura de gas correcta
	gasOK    int       // redes con gas OK en la última lectura
	gasTotal int       // redes consultadas en la última lectura

	// Notificación transitoria (toast) en el footer.
	notice      string
	noticeLevel noticeLevel
	noticeUntil time.Time

	// allNetworks es el catálogo completo de redes (con overrides de config);
	// `networks` es la vista filtrada por las redes activas del usuario, que es lo
	// que consumen balances, gas y transacciones.
	allNetworks []chain.Network

	// Coordinación de cargas async (3.6): loadGen etiqueta cada carga; los
	// resultados con una generación vieja se descartan. loadCancel cancela el
	// trabajo de red en vuelo al iniciar una carga nueva o cambiar de contexto.
	loadGen    int
	loadCancel context.CancelFunc

	spinner        spinner.Model
	active         tab
	helpOpen       bool // overlay de ayuda (?) visible
	debugOpen      bool // overlay de métricas/debug (ctrl+g) visible (3.3)
	networksOpen   bool // overlay de selección de redes (n) visible
	networksCursor int
	width          int
	height         int

	// Pantalla de bienvenida (portada con logo + intro). showWelcome es si está
	// visible ahora; welcomeHide es el estado del conmutador "no volver a mostrar";
	// prefs persiste esa preferencia entre sesiones.
	prefs       *storage.Prefs
	showWelcome bool
	welcomeHide bool
	// contentW/contentH son el área útil interior del frame, recalculada en cada
	// WindowSizeMsg para dimensionar el contenido y los modales.
	contentW int
	contentH int

	// Pestaña Cuentas
	input         textinput.Model
	addErr        error
	accCursor     int
	resolvingName string // nombre ENS que se está resolviendo para añadir, si hay
	// Confirmación de borrado (2.8): el primer ctrl+d arma la confirmación sobre
	// confirmDelAddr; el segundo (misma wallet) borra. esc o mover el cursor cancela.
	confirmDel     bool
	confirmDelAddr common.Address
	// Detalle de wallet (2.7): modal con el agregado de la wallet seleccionada
	// entre redes. Reutiliza txViewport (solo uno abierto a la vez, en otra tab).
	walletDetailOpen bool

	// Paleta de comandos (2.2): overlay global de búsqueda difusa de acciones.
	paletteOpen   bool
	paletteInput  textinput.Model
	paletteCursor int

	// Tour de primera vez (2.10): recorrido guiado tras la portada.
	tourActive bool
	tourStep   int

	// Pestaña Balances
	balState   loadState
	balResults []chain.BalanceResult
	balCursor  int
	balFocus   bool // true = solo la wallet seleccionada; false = todas
	// Ordenación (2.3): clave (0=catálogo, 1=valor, 2=red, 3=wallet) y dirección.
	balSortKey int
	balSortAsc bool

	// Pestaña Transacciones (historial multi-red por wallet)
	txState      loadState
	txs          []txRow // fusionadas de todas las redes, ordenadas desc
	txErr        error
	txCursor     int // índice sobre la lista visible (filtrada)
	txScroll     int // desplazamiento de la ventana visible
	txWallet     common.Address
	txNetFilter  uint64          // 0 = todas las redes activas; si no, una red
	txPage       map[uint64]int  // última página cargada por red
	txExhausted  map[uint64]bool // redes sin más páginas
	txDetailOpen bool            // modal de detalle de la tx seleccionada
	txViewport   viewport.Model  // contenido scrollable del modal
	// Ordenación de txs (2.3): clave (0=fecha, 1=red, 2=valor) y dirección.
	txSortKey int
	txSortAsc bool
}

// noticeLevel clasifica el tono de un toast del footer.
type noticeLevel int

const (
	noticeInfo noticeLevel = iota
	noticeError
)

// noticeTTL es cuánto se muestra un toast antes de desaparecer.
const noticeTTL = 4 * time.Second

// noticeClearMsg pide borrar el toast; `at` identifica la notice concreta para no
// borrar una más nueva fijada entretanto.
type noticeClearMsg struct{ at time.Time }

// setNotice fija un toast en el footer. El llamador debe lanzar noticeClearCmd
// para que desaparezca solo.
func (m *Model) setNotice(level noticeLevel, text string) {
	m.notice = text
	m.noticeLevel = level
	m.noticeUntil = time.Now().Add(noticeTTL)
}

// noticeClearCmd programa el borrado del toast tras noticeTTL.
func noticeClearCmd(at time.Time) tea.Cmd {
	return tea.Tick(noticeTTL, func(time.Time) tea.Msg { return noticeClearMsg{at: at} })
}

// NewModel construye el modelo raíz inyectando todas las dependencias: cliente
// de cadena, almacenamiento de wallets, redes efectivas (con overrides de config)
// e intervalo de refresco.
func NewModel(client *chain.Client, wallets *storage.Wallets, networks []chain.Network, refresh time.Duration, txProvider chain.TxProvider, ens *chain.ENSResolver, prices chain.PriceProvider, fiatCurrency string, tokens chain.TokenBalanceProvider, theme string, prefs *storage.Prefs) Model {
	// El tema guardado en prefs (cambio en caliente) tiene prioridad sobre la
	// config TOML. Con "auto" arrancamos en oscuro y ajustamos al detectar el fondo.
	themePref := theme
	if prefs != nil && prefs.Theme != "" {
		themePref = prefs.Theme
	}
	themeNow := themePref
	if themeNow == themeAuto {
		themeNow = themeDark
	}
	styles := StylesFor(paletteByName(themeNow))
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

	// Input de la paleta de comandos (2.2).
	pi := textinput.New()
	pi.Placeholder = "Buscar comando…"
	pi.Prompt = "› "
	pi.CharLimit = 48

	// El tour (2.10) lo lanza la portada al cerrarse; si la portada está oculta
	// (usuario que la desactivó), arranca directo en el primer uso pendiente.
	showWelcome := prefs == nil || !prefs.HideWelcome
	startTourNow := !showWelcome && prefs != nil && !prefs.TourDone

	return Model{
		styles:        styles,
		client:        client,
		wallets:       wallets,
		networks:      enabledNetworks(networks, prefs),
		allNetworks:   networks,
		refresh:       refresh,
		txProvider:    txProvider,
		ens:           ens,
		priceProvider: prices,
		fiatCurrency:  fiatCurrency,
		prices:        make(map[chain.PriceQuery]float64),
		tokenProvider: tokens,
		themePref:     themePref,
		themeNow:      themeNow,
		prefs:         prefs,
		// Mostramos la portada al arrancar salvo que el usuario pidiera ocultarla.
		showWelcome:  showWelcome,
		welcomeHide:  prefs != nil && prefs.HideWelcome,
		tourActive:   startTourNow,
		ensNames:     make(map[common.Address]string),
		gas:          make(map[uint64]*big.Int),
		gasPrev:      make(map[uint64]*big.Int),
		spinner:      sp,
		input:        ti,
		txViewport:   vp,
		paletteInput: pi,
		active:       tabAccounts,
		txPage:       map[uint64]int{},
		txExhausted:  map[uint64]bool{},
	}
}

// WithDebug arranca con el overlay de métricas/debug visible (flag --debug). Es
// un setter encadenable para no alargar más la firma de NewModel.
func (m Model) WithDebug(on bool) Model {
	m.debugOpen = on
	return m
}

// Init arranca los bucles de refresco (balances y gas), una primera lectura de
// gas inmediata y la resolución ENS de las wallets ya seguidas.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		refreshTickCmd(m.refresh),
		gasTickCmd(m.refresh),
		m.fetchGasCmd(),
		m.resolveWalletNamesCmd(),
		tea.RequestBackgroundColor, // tema "auto": detectar el fondo del terminal
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
			m.paletteInput.SetWidth(m.contentW - 12)
		}
		// El modal de detalle de tx vive en el área de contenido; le dejamos 2
		// líneas para su propia pista de ayuda.
		m.txViewport.SetWidth(m.contentW)
		if m.contentH > 2 {
			m.txViewport.SetHeight(m.contentH - 2)
		}

	case tea.KeyPressMsg:
		// ctrl+c siempre sale, pase lo que pase.
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// La portada de bienvenida captura el teclado mientras está visible: enter
		// entra a la app, 'd' conmuta "no volver a mostrar" (se guarda al entrar),
		// 'q' sale.
		if m.showWelcome {
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "d":
				m.welcomeHide = !m.welcomeHide
			case "enter", " ", "esc":
				m.showWelcome = false
				if m.prefs != nil && m.welcomeHide != m.prefs.HideWelcome {
					_ = m.prefs.SetHideWelcome(m.welcomeHide)
				}
				m.maybeStartTour() // primer arranque: lanzar el tour guiado (2.10)
			}
			return m, nil
		}
		// El tour de primera vez (2.10) captura el teclado mientras está activo.
		if m.tourActive {
			return m.updateTour(msg)
		}
		// La paleta de comandos (2.2) captura el teclado mientras está abierta.
		if m.paletteOpen {
			return m.updatePalette(msg)
		}
		// El overlay de selección de redes captura el teclado mientras está abierto.
		if m.networksOpen {
			return m.updateNetworks(msg)
		}
		// El overlay de ayuda tiene prioridad sobre todo lo demás: mientras está
		// abierto, '?'/esc/q lo cierran y el resto de teclas se ignoran.
		if m.helpOpen {
			switch msg.String() {
			case "?", "esc", "q":
				m.helpOpen = false
			}
			return m, nil
		}
		if msg.String() == "?" {
			m.helpOpen = true
			return m, nil
		}
		// El overlay de debug (3.3) se conmuta con ctrl+g; mientras está abierto,
		// ctrl+g/esc/q lo cierran y el resto de teclas se ignoran.
		if m.debugOpen {
			switch msg.String() {
			case "ctrl+g", "esc", "q":
				m.debugOpen = false
			}
			return m, nil
		}
		if msg.String() == "ctrl+g" {
			m.debugOpen = true
			return m, nil
		}
		// ctrl+k abre la paleta de comandos (global, no choca con el input porque no
		// es una letra). Cerrar las modales/overlay activos antes.
		if msg.String() == "ctrl+k" {
			m.paletteOpen = true
			m.txDetailOpen = false
			m.walletDetailOpen = false
			m.paletteInput.Reset()
			m.paletteInput.Focus()
			m.paletteCursor = 0
			return m, nil
		}
		// 'n' abre la selección de redes. Solo fuera de Cuentas (allí 'n' se escribe
		// en el input, p.ej. nombres ENS) y sin el modal de detalle abierto.
		if msg.String() == "n" && m.active != tabAccounts && !m.txDetailOpen {
			m.networksOpen = true
			m.networksCursor = 0
			return m, nil
		}

		switch msg.String() {
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
		// Descartamos resultados de una carga ya superada (3.6).
		if msg.gen != m.loadGen {
			return m, nil
		}
		m.balResults = msg.results
		m.balState = stateLoaded
		m.clampBalCursor()
		// Tasamos en fiat lo que acabamos de cargar (1.1), como nueva generación.
		ctx, gen := m.nextLoad()
		cmds := []tea.Cmd{m.fetchPricesCmd(ctx, gen)}
		if n := countBalanceErrors(msg.results); n > 0 {
			m.setNotice(noticeError, fmt.Sprintf("⚠ %d balance(s) no se cargaron", n))
			cmds = append(cmds, noticeClearCmd(m.noticeUntil))
		}
		return m, tea.Batch(cmds...)

	case pricesMsg:
		if msg.gen != m.loadGen {
			return m, nil
		}
		for q, price := range msg.prices {
			m.prices[q] = price
		}
		return m, nil

	case txPageMsg:
		// Descartamos resultados de una carga ya superada (cambio de wallet, red o
		// pestaña avanzan la generación) (3.6).
		if msg.gen != m.loadGen || msg.wallet != m.txWallet {
			return m, nil
		}
		m.txState = stateLoaded
		var firstErr error
		okCount := 0
		for _, r := range msg.results {
			if r.err != nil {
				if firstErr == nil {
					firstErr = r.err
				}
				continue
			}
			okCount++
			m.txPage[r.chainID] = r.page
			if len(r.rows) < txPageSize {
				m.txExhausted[r.chainID] = true
			}
			m.txs = append(m.txs, r.rows...)
		}
		m.txs = sortDedupTxRows(m.txs)
		m.clampTxCursor()
		// Si todas las redes fallaron y no hay nada en pantalla, es un error de
		// verdad; si solo fallaron algunas, avisamos con un toast sin romper.
		if okCount == 0 && firstErr != nil && len(m.txs) == 0 {
			m.txErr = firstErr
			m.txState = stateError
			return m, nil
		}
		m.txErr = nil
		if firstErr != nil {
			m.setNotice(noticeError, "⚠ algunas redes fallaron al cargar txs")
			return m, noticeClearCmd(m.noticeUntil)
		}
		return m, nil

	case exportDoneMsg:
		switch {
		case msg.err != nil:
			m.setNotice(noticeError, "No se pudo exportar: "+msg.err.Error())
		case msg.count == 0:
			m.setNotice(noticeInfo, "Nada que exportar")
		default:
			m.setNotice(noticeInfo, fmt.Sprintf("✓ %d filas exportadas a %s", msg.count, msg.path))
		}
		return m, noticeClearCmd(m.noticeUntil)

	case copiedMsg:
		if msg.err != nil {
			m.setNotice(noticeError, "No se pudo copiar: "+msg.err.Error())
		} else {
			m.setNotice(noticeInfo, "✓ "+msg.label+" copiado al portapapeles")
		}
		return m, noticeClearCmd(m.noticeUntil)

	case openedMsg:
		if msg.err != nil {
			m.setNotice(noticeError, "No se pudo abrir el navegador: "+msg.err.Error())
			return m, noticeClearCmd(m.noticeUntil)
		}
		return m, nil

	case ensResolvedMsg:
		for addr, name := range msg.names {
			m.ensNames[addr] = name
		}
		return m, nil

	case noticeClearMsg:
		// Solo borramos si nadie fijó una notice más nueva entretanto.
		if m.noticeUntil.Equal(msg.at) {
			m.notice = ""
		}
		return m, nil

	case tea.BackgroundColorMsg:
		// Tema "auto": ajustamos la paleta al fondo real del terminal. Con un tema
		// explícito (dark/light) ignoramos la detección.
		if m.themePref == themeAuto {
			if msg.IsDark() {
				m.applyTheme(themeDark)
			} else {
				m.applyTheme(themeLight)
			}
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
		ok := 0
		for _, r := range msg.results {
			if r.Err != nil || r.Wei == nil {
				continue // conservamos el último valor bueno de esa red
			}
			ok++
			if cur, exists := m.gas[r.ChainID]; exists {
				m.gasPrev[r.ChainID] = cur // el actual pasa a ser el anterior
			}
			m.gas[r.ChainID] = r.Wei
		}
		m.gasOK, m.gasTotal = ok, len(msg.results)
		m.lastGas = time.Now()
		if rl := m.client.RateLimitedChains(); len(rl) > 0 {
			m.setNotice(noticeError, fmt.Sprintf("⚠ %d red(es) rate-limited · sirviendo caché", len(rl)))
			return m, noticeClearCmd(m.noticeUntil)
		}
		return m, nil

	case refreshTickMsg:
		// El tick siempre se reprograma; solo refrescamos si estamos viendo
		// Balances, hay wallets y no hay ya una carga en vuelo (anti-solape).
		cmds := []tea.Cmd{refreshTickCmd(m.refresh)}
		if m.active == tabBalances && m.balState != stateLoading && m.wallets.Len() > 0 {
			m.balState = stateLoading
			ctx, gen := m.nextLoad()
			cmds = append(cmds, m.spinner.Tick, m.fetchBalancesCmd(ctx, gen))
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

// nextLoad invalida la carga anterior (cancela su contexto en vuelo y avanza la
// generación, para descartar resultados que lleguen tarde) y devuelve un contexto
// nuevo y la generación para la siguiente carga (3.6).
func (m *Model) nextLoad() (context.Context, int) {
	if m.loadCancel != nil {
		m.loadCancel()
	}
	m.loadGen++
	ctx, cancel := context.WithCancel(context.Background())
	m.loadCancel = cancel
	return ctx, m.loadGen
}

// cancelLoad cancela el trabajo de red en vuelo y avanza la generación sin iniciar
// una carga nueva. Lo usan los cambios de contexto que no recargan de inmediato
// (p.ej. navegar a Cuentas), para no aplicar resultados ya obsoletos.
func (m *Model) cancelLoad() {
	if m.loadCancel != nil {
		m.loadCancel()
		m.loadCancel = nil
	}
	m.loadGen++
}

// onEnterTab ajusta el foco del input y lanza la carga de balances al entrar en
// la pestaña Balances por primera vez.
func (m *Model) onEnterTab() tea.Cmd {
	m.txDetailOpen = false // cambiar de pestaña cierra los modales de detalle
	m.walletDetailOpen = false
	switch m.active {
	case tabAccounts:
		m.cancelLoad() // navegar a Cuentas cancela cualquier carga en vuelo (3.6)
		return m.input.Focus()
	case tabBalances:
		m.input.Blur()
		if m.balState == stateIdle && m.wallets.Len() > 0 {
			m.balState = stateLoading
			ctx, gen := m.nextLoad()
			return tea.Batch(m.spinner.Tick, m.fetchBalancesCmd(ctx, gen))
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

// applyTheme cambia la paleta efectiva en caliente: reconstruye los estilos y el
// del spinner (que cachea el suyo aparte).
func (m *Model) applyTheme(name string) {
	m.themeNow = name
	m.styles = StylesFor(paletteByName(name))
	m.spinner.Style = m.styles.Spinner
}

// cycleTheme alterna explícitamente entre claro y oscuro (deja de ser "auto") y
// persiste la elección en prefs. Lo usa el command palette (2.2).
func (m *Model) cycleTheme() {
	next := themeDark
	if m.themeNow == themeDark {
		next = themeLight
	}
	m.themePref = next
	m.applyTheme(next)
	if m.prefs != nil {
		_ = m.prefs.SetTheme(next)
	}
}

// View dibuja toda la app como un frame que llena el terminal (ver layout.go).
// AltScreen pone la TUI en el búfer alterno (pantalla completa): sin él, un frame
// de altura completa hace scroll y deja el header fuera de vista.
func (m Model) View() tea.View {
	v := tea.NewView(m.renderFrame())
	v.AltScreen = true
	v.WindowTitle = "chainview"
	return v
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
