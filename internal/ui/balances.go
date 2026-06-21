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
		if m.balCursor < len(m.balResults)-1 {
			m.balCursor++
		}
	case "r":
		if m.balState != stateLoading && m.wallets.Len() > 0 {
			m.balState = stateLoading
			return m, tea.Batch(m.spinner.Tick, m.fetchBalancesCmd())
		}
	}
	return m, nil
}

func (m *Model) clampBalCursor() {
	if m.balCursor >= len(m.balResults) {
		m.balCursor = len(m.balResults) - 1
	}
	if m.balCursor < 0 {
		m.balCursor = 0
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

	var b strings.Builder
	b.WriteString(m.styles.Faint.Render(fit("Wallet", 16) + fit("Red", 14) + "Balance"))
	b.WriteString("\n")
	if m.balState == stateLoading {
		// Refresco con datos ya en pantalla.
		b.WriteString(m.styles.Faint.Render(m.spinner.View()+" actualizando…") + "\n")
	}

	for i, r := range m.balResults {
		wallet := fit(m.displayName(r.Address), 16)
		red := fit(m.networkName(r.ChainID), 14)

		var bal string
		switch {
		case r.Err != nil:
			bal = "error"
		default:
			bal = chain.FormatEther(r.Wei) + " " + m.networkSymbol(r.ChainID)
		}

		switch {
		case i == m.balCursor:
			b.WriteString(m.styles.Balance.Render("› " + wallet + red + bal))
		case r.Err != nil:
			b.WriteString("  " + wallet + red + m.styles.Error.Render(bal))
		default:
			b.WriteString("  " + wallet + red + bal)
		}
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
