package ui

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/ethereum/go-ethereum/common"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
)

// balancesMsg transporta el resultado del fetch concurrente de balances.
type balancesMsg struct {
	results []chain.BalanceResult
}

// refreshTickMsg lo emite el tick periódico de refresco.
type refreshTickMsg struct{}

// refreshTickCmd programa el siguiente tick de refresco (decisión D2: polling).
func refreshTickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return refreshTickMsg{} })
}

// fetchBalancesCmd lanza el fetch concurrente (wallets × redes) en background.
// Captura las dependencias por valor para no cerrar sobre el Model completo.
func (m Model) fetchBalancesCmd() tea.Cmd {
	client := m.client
	addrs := m.wallets.List()
	networks := m.networks
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return balancesMsg{results: client.FetchAll(ctx, addrs, networks)}
	}
}

// updateBalances maneja las teclas de la pestaña Balances: navegar la tabla y
// recargar manualmente.
func (m Model) updateBalances(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		if m.balCursor > 0 {
			m.balCursor--
		}
	case "down":
		if m.balCursor < len(m.visibleBalances())-1 {
			m.balCursor++
		}
	case "f":
		// Alterna entre ver todas las wallets y solo la seleccionada en Cuentas.
		m.balFocus = !m.balFocus
		m.balCursor = 0
	case "e":
		return m, m.exportBalancesCmd()
	case "r":
		if m.balState != stateLoading && m.wallets.Len() > 0 {
			m.balState = stateLoading
			return m, tea.Batch(m.spinner.Tick, m.fetchBalancesCmd())
		}
	}
	return m, nil
}

// visibleBalances son los balances que se muestran: todos, o solo los de la
// wallet seleccionada cuando está activo el foco individual.
func (m Model) visibleBalances() []chain.BalanceResult {
	if !m.balFocus {
		return m.balResults
	}
	wallet, ok := m.selectedWallet()
	if !ok {
		return m.balResults
	}
	out := make([]chain.BalanceResult, 0, len(m.networks))
	for _, r := range m.balResults {
		if r.Address == wallet {
			out = append(out, r)
		}
	}
	return out
}

// countBalanceErrors cuenta las celdas con error en una tanda de balances.
func countBalanceErrors(results []chain.BalanceResult) int {
	n := 0
	for _, r := range results {
		if r.Err != nil {
			n++
		}
	}
	return n
}

func (m *Model) clampBalCursor() {
	if m.balCursor >= len(m.visibleBalances()) {
		m.balCursor = len(m.visibleBalances()) - 1
	}
	if m.balCursor < 0 {
		m.balCursor = 0
	}
}

// balanceColumns define las columnas de la tabla de balances. El espaciador flex
// empuja el balance (número + símbolo) contra el borde derecho, de modo que los
// importes queden alineados a la derecha y sean fáciles de comparar.
func balanceColumns() []column {
	return []column{
		{title: "Wallet", align: alignLeft, min: 16},
		{title: "Red", align: alignLeft, min: 12},
		{title: "", align: alignLeft, min: 1, flex: true}, // espaciador
		{title: "Balance", align: alignRight, min: 12},
		{title: "", align: alignLeft, min: 5}, // símbolo
	}
}

func (m Model) renderBalances() string {
	if m.wallets.Len() == 0 {
		return m.renderState("◯", "Sin wallets", "Añádelas en la pestaña Cuentas para ver balances")
	}
	// Primera carga sin datos previos: estado de carga centrado.
	if m.balState == stateLoading && len(m.balResults) == 0 {
		return m.renderState(m.spinner.View(), "Cargando balances…", "")
	}

	cols := balanceColumns()
	widths := layoutColumns(cols, m.contentW)
	vis := m.visibleBalances()

	// Contexto: modo de vista (todas / wallet seleccionada).
	scope := "todas las wallets"
	if m.balFocus {
		if w, ok := m.selectedWallet(); ok {
			scope = m.displayName(w)
		}
	}
	ctx := m.styles.Balance.Render("Saldos") + m.styles.Faint.Render(" · "+scope)
	if m.balState == stateLoading {
		ctx += m.styles.Faint.Render("  " + m.spinner.View())
	}

	var b strings.Builder
	b.WriteString(ctx + "\n\n")
	b.WriteString(m.tableHeader(cols, widths) + "\n")
	b.WriteString(m.tableRule(widths) + "\n")

	for i, r := range vis {
		wallet := m.displayName(r.Address)
		red := m.networkName(r.ChainID)

		// Importe a la derecha + símbolo como dato secundario; en error, un guion
		// neutro y "error" en rojo en la columna del símbolo.
		amount, symbol := chain.FormatEther(r.Wei), m.networkSymbol(r.ChainID)
		amountCell := styledCell(amount, m.styles.Balance)
		symbolCell := styledCell(symbol, m.styles.Symbol)
		if r.Err != nil {
			amountCell = styledCell("—", m.styles.Faint)
			symbolCell = styledCell("error", m.styles.Error)
		}

		cells := []tcell{
			txt(wallet),
			styledCell(red, m.styles.Faint),
			txt(""), // espaciador
			amountCell,
			symbolCell,
		}
		b.WriteString(m.tableRow(cols, widths, cells, i == m.balCursor))
		b.WriteString("\n")
	}
	return b.String()
}

// networkName resuelve el nombre legible de una red por su chain ID.
func (m Model) networkName(chainID uint64) string {
	for _, n := range m.networks {
		if n.ChainID == chainID {
			return n.Name
		}
	}
	return "?"
}

// networkSymbol resuelve el símbolo de la moneda nativa de una red.
func (m Model) networkSymbol(chainID uint64) string {
	for _, n := range m.networks {
		if n.ChainID == chainID {
			return n.Symbol
		}
	}
	return ""
}

// shortAddr acorta una dirección a la forma 0x1234…abcd.
func shortAddr(a common.Address) string {
	h := a.Hex()
	return h[:6] + "…" + h[len(h)-4:]
}

// fit ajusta un string a un ancho fijo: trunca con … si sobra, o rellena con
// espacios si falta. Sirve para alinear columnas de la tabla.
func fit(s string, w int) string {
	r := []rune(s)
	if len(r) > w {
		if w <= 1 {
			return string(r[:w])
		}
		return string(r[:w-1]) + "…"
	}
	return s + strings.Repeat(" ", w-len(r))
}
