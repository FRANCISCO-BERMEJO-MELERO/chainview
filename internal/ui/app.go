package ui

import (
	"context"
	"math/big"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ethereum/go-ethereum/common"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/ui/assets"
)

// demoAddress es una wallet conocida de mainnet (vitalik.eth) que usamos en la
// Semana 3 para mostrar un balance real antes de tener gestión de cuentas. En la
// Semana 4 se sustituye por las wallets que el usuario añada.
var demoAddress = common.HexToAddress("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")

// loadState modela el ciclo de vida de una carga asíncrona.
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

// orderedTabs fija el orden de navegación de las pestañas.
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

func (t tab) placeholder() string {
	switch t {
	case tabAccounts:
		return "Aquí irán las wallets seguidas.\nPróximamente: añadir y eliminar cuentas."
	case tabTransactions:
		return "Aquí irá el historial de transacciones.\nPróximamente: últimas txs por wallet."
	default:
		return ""
	}
}

// Model es el modelo raíz de la TUI (patrón Elm: estado + Init/Update/View).
type Model struct {
	styles  Styles
	client  *chain.Client
	spinner spinner.Model

	active tab
	width  int
	height int

	// Estado del balance de demostración (Semana 3: una wallet, una red).
	// La Semana 4 lo reemplaza por una tabla wallet × red.
	balanceState loadState
	balance      *big.Int
	balanceErr   error
}

// NewModel construye el modelo raíz inyectando el chain.Client (wiring): así los
// tea.Cmd del modelo tienen un cliente al que pedir datos. En semanas posteriores
// este constructor recibirá también config y storage.
func NewModel(client *chain.Client) Model {
	styles := DefaultStyles()
	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(styles.Spinner),
	)
	return Model{
		styles:  styles,
		client:  client,
		spinner: sp,
		active:  tabAccounts,
	}
}

// Init no dispara ningún comando inicial: la carga del balance se lanza al entrar
// en la pestaña Balances.
func (m Model) Init() tea.Cmd {
	return nil
}

// balanceLoadedMsg / balanceErrMsg son los resultados del fetch asíncrono de
// balance. Update los procesa para actualizar el Model.
type balanceLoadedMsg struct {
	chainID uint64
	addr    common.Address
	wei     *big.Int
}

type balanceErrMsg struct {
	chainID uint64
	addr    common.Address
	err     error
}

// fetchBalanceCmd devuelve un tea.Cmd que consulta el balance en background y
// emite un msg con el resultado. El RPC va dentro de la goroutine del Cmd (nunca
// en Update) y con timeout, para no congelar la TUI ni colgar la goroutine.
func fetchBalanceCmd(c *chain.Client, chainID uint64, addr common.Address) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		wei, err := c.BalanceAt(ctx, chainID, addr)
		if err != nil {
			return balanceErrMsg{chainID: chainID, addr: addr, err: err}
		}
		return balanceLoadedMsg{chainID: chainID, addr: addr, wei: wei}
	}
}

// Update procesa los mensajes entrantes y actualiza el modelo.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "right", "l":
			m.active = m.nextTab(1)
			return m, m.maybeFetchBalance()
		case "shift+tab", "left", "h":
			m.active = m.nextTab(-1)
			return m, m.maybeFetchBalance()
		case "r":
			// Reintentar la carga del balance (útil con RPC públicos lentos).
			if m.active == tabBalances {
				m.balanceState = stateIdle
				return m, m.maybeFetchBalance()
			}
		}

	case balanceLoadedMsg:
		m.balance = msg.wei
		m.balanceErr = nil
		m.balanceState = stateLoaded
		return m, nil

	case balanceErrMsg:
		m.balanceErr = msg.err
		m.balanceState = stateError
		return m, nil

	case spinner.TickMsg:
		// Solo mantenemos vivo el spinner mientras de verdad estamos cargando:
		// al terminar dejamos morir el tick para no gastar CPU.
		if m.balanceState != stateLoading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// maybeFetchBalance lanza la carga del balance la primera vez que se entra en la
// pestaña Balances. No re-consulta si ya está cargando o cargado. Devuelve el
// Cmd combinado del spinner + el fetch (o nil si no hay nada que hacer).
//
// Usa receptor por puntero porque muta balanceState; quien llama debe invocarla
// y guardar el Cmd ANTES de devolver m, para que el cambio de estado viaje en el
// Model devuelto.
func (m *Model) maybeFetchBalance() tea.Cmd {
	if m.active != tabBalances || m.balanceState != stateIdle {
		return nil
	}
	m.balanceState = stateLoading
	return tea.Batch(
		m.spinner.Tick,
		fetchBalanceCmd(m.client, chain.ChainEthereum, demoAddress),
	)
}

// nextTab devuelve la pestaña desplazada delta posiciones, con envoltura circular.
func (m Model) nextTab(delta int) tab {
	n := len(orderedTabs)
	return tab((int(m.active) + delta + n) % n)
}

// View renderiza la TUI completa: título + barra de pestañas + cuerpo + ayuda.
func (m Model) View() tea.View {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render(assets.Title))
	b.WriteString("\n\n")
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")
	b.WriteString(m.renderBody())
	b.WriteString("\n\n")
	b.WriteString(m.styles.Faint.Render("tab / ← → cambiar pestaña · r recargar · q salir"))

	return tea.NewView(b.String())
}

// renderTabs dibuja la barra de pestañas resaltando la activa.
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

// renderBody dibuja el panel de la pestaña activa, adaptando el ancho al terminal.
func (m Model) renderBody() string {
	var content string
	if m.active == tabBalances {
		content = m.renderBalances()
	} else {
		content = m.active.placeholder()
	}

	panel := m.styles.Panel
	if m.width > 0 {
		panel = panel.Width(m.width - 6)
	}
	return panel.Render(content)
}

// renderBalances muestra el balance de la wallet de demo en Ethereum según el
// estado de la carga (cargando / cargado / error).
func (m Model) renderBalances() string {
	addr := demoAddress.Hex()
	short := addr[:6] + "…" + addr[len(addr)-4:]
	header := "Ethereum   " + short + "\n\n"

	switch m.balanceState {
	case stateLoading:
		return header + m.spinner.View() + " consultando balance…"
	case stateError:
		return header + m.styles.Error.Render("error: "+m.balanceErr.Error())
	case stateLoaded:
		return header + m.styles.Balance.Render(chain.FormatEther(m.balance)+" ETH")
	default:
		return header + m.styles.Faint.Render("entra en esta pestaña para cargar el balance…")
	}
}
