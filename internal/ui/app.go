package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/ui/assets"
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

// title es el nombre visible de cada pestaña.
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

// placeholder es el contenido provisional de cada pestaña hasta que se
// implementen las vistas reales (cuentas, balances y transacciones).
func (t tab) placeholder() string {
	switch t {
	case tabAccounts:
		return "Aquí irán las wallets seguidas.\nPróximamente: añadir y eliminar cuentas."
	case tabBalances:
		return "Aquí irá la tabla de balances por wallet y red.\nPróximamente: balances reales on-chain."
	case tabTransactions:
		return "Aquí irá el historial de transacciones.\nPróximamente: últimas txs por wallet."
	default:
		return ""
	}
}

// Model es el modelo raíz de la TUI (patrón Elm: estado + Init/Update/View).
type Model struct {
	styles Styles
	active tab
	width  int
	height int
}

// NewModel construye el modelo raíz con el tema por defecto y la primera
// pestaña seleccionada. A partir de semanas posteriores este constructor
// recibirá también el chain.Client, config y storage (ver tarea S3-6).
func NewModel() Model {
	return Model{
		styles: DefaultStyles(),
		active: tabAccounts,
	}
}

// Init no dispara ningún comando inicial todavía.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update procesa los mensajes entrantes y actualiza el modelo.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Guardamos el tamaño del terminal para que las vistas se adapten al
		// ancho/alto disponible (base para layouts responsivos).
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "right", "l":
			m.active = m.nextTab(1)
		case "shift+tab", "left", "h":
			m.active = m.nextTab(-1)
		}
	}
	return m, nil
}

// nextTab devuelve la pestaña desplazada delta posiciones, con envoltura
// circular (de la última vuelve a la primera y viceversa).
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
	b.WriteString(m.styles.Faint.Render("tab / ← → cambiar pestaña · q salir"))

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

// renderBody dibuja el panel de la pestaña activa, adaptando el ancho al
// terminal cuando ya conocemos su tamaño (tras el primer WindowSizeMsg).
func (m Model) renderBody() string {
	panel := m.styles.Panel
	if m.width > 0 {
		// Restamos los bordes/padding horizontales del panel (2+2 padding, 1+1 borde).
		panel = panel.Width(m.width - 6)
	}
	return panel.Render(m.active.placeholder())
}
