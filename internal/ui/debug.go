package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// renderDebug dibuja el overlay de métricas/modo debug (3.3): contadores de la
// caché RPC, estado de redes y datos de la sesión. Es de solo lectura; se conmuta
// con ctrl+g (o arranca abierto con --debug).
func (m Model) renderDebug() string {
	var b strings.Builder
	b.WriteString(m.styles.StateTitle.Render("⚙  Debug / metrics"))
	b.WriteString("\n")

	row := func(k, v string) {
		b.WriteString("  " + m.styles.Brand.Render(fit(k, 18)) + m.styles.Faint.Render(v) + "\n")
	}
	section := func(title string) {
		b.WriteString("\n" + m.styles.Balance.Render(title) + "\n")
	}

	section("RPC cache")
	if m.client != nil {
		s := m.client.Stats()
		total := s.CacheHits + s.RPCCalls
		ratio := "—"
		if total > 0 {
			ratio = fmt.Sprintf("%.0f%%", float64(s.CacheHits)*100/float64(total))
		}
		row("RPC calls", fmt.Sprintf("%d", s.RPCCalls))
		row("Cache hits", fmt.Sprintf("%d  (%s)", s.CacheHits, ratio))
		row("429 rate-limit", fmt.Sprintf("%d", s.RateLimitHits))
		row("Stale served", fmt.Sprintf("%d", s.StaleServed))
		if rl := m.client.RateLimitedChains(); len(rl) > 0 {
			row("Networks in cooldown", fmt.Sprintf("%v", rl))
		}
	} else {
		row("(no client)", "—")
	}

	section("Session")
	row("Active networks", fmt.Sprintf("%d / %d", len(m.networks), len(m.allNetworks)))
	row("Wallets", fmt.Sprintf("%d", m.wallets.Len()))
	row("Load generation", fmt.Sprintf("%d", m.loadGen))
	row("Theme", m.themeNow)
	row("Terminal", fmt.Sprintf("%d×%d  (content %d×%d)", m.width, m.height, m.contentW, m.contentH))

	b.WriteString(m.styles.Faint.Render("ctrl+g or esc to close"))

	panel := m.styles.Panel.Render(strings.TrimRight(b.String(), "\n"))
	// Clamp por si el panel no cabe en terminales bajos (no debe desbordar el frame).
	panel = lipgloss.NewStyle().MaxWidth(m.contentW).MaxHeight(m.contentH).Render(panel)
	return lipgloss.Place(m.contentW, m.contentH, lipgloss.Center, lipgloss.Center, panel)
}
