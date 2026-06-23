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
	b.WriteString(m.styles.StateTitle.Render("⚙  Debug / métricas"))
	b.WriteString("\n")

	row := func(k, v string) {
		b.WriteString("  " + m.styles.Brand.Render(fit(k, 18)) + m.styles.Faint.Render(v) + "\n")
	}
	section := func(title string) {
		b.WriteString("\n" + m.styles.Balance.Render(title) + "\n")
	}

	section("Caché RPC")
	if m.client != nil {
		s := m.client.Stats()
		total := s.CacheHits + s.RPCCalls
		ratio := "—"
		if total > 0 {
			ratio = fmt.Sprintf("%.0f%%", float64(s.CacheHits)*100/float64(total))
		}
		row("Llamadas RPC", fmt.Sprintf("%d", s.RPCCalls))
		row("Hits de caché", fmt.Sprintf("%d  (%s)", s.CacheHits, ratio))
		row("429 rate-limit", fmt.Sprintf("%d", s.RateLimitHits))
		row("Valores viejos", fmt.Sprintf("%d", s.StaleServed))
		if rl := m.client.RateLimitedChains(); len(rl) > 0 {
			row("Redes en cooldown", fmt.Sprintf("%v", rl))
		}
	} else {
		row("(sin cliente)", "—")
	}

	section("Sesión")
	row("Redes activas", fmt.Sprintf("%d / %d", len(m.networks), len(m.allNetworks)))
	row("Wallets", fmt.Sprintf("%d", m.wallets.Len()))
	row("Tema", m.themeNow)
	row("Terminal", fmt.Sprintf("%d×%d  (contenido %d×%d)", m.width, m.height, m.contentW, m.contentH))

	b.WriteString("\n" + m.styles.Faint.Render("ctrl+g o esc para cerrar"))

	panel := m.styles.Panel.Render(strings.TrimRight(b.String(), "\n"))
	return lipgloss.Place(m.contentW, m.contentH, lipgloss.Center, lipgloss.Center, panel)
}
