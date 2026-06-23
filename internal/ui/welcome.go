package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/ui/assets"
)

// renderWelcome dibuja la portada de bienvenida: logo + intro + comandos básicos
// + el conmutador de "no volver a mostrar". Es responsiva en alto: si no caben
// todas las secciones, descarta primero la intro y luego los comandos, de modo
// que el bloque nunca desborde el frame (que en terminales pequeños es escaso).
func (m Model) renderWelcome() string {
	avail := m.height - 2 // alto interior del frame (sin el borde)
	if avail < 1 {
		avail = 1
	}

	// Logo grande si cabe a lo ancho; si no, la marca compacta de una línea.
	bigLogo := strings.Join(assets.LogoLines(), "\n")
	useBig := lipgloss.Width(bigLogo) <= m.contentW
	logo := assets.LogoCompact
	if useBig {
		logo = bigLogo
	}
	logo = m.styles.Brand.Render(logo)

	tagline := m.styles.Balance.Render("Watch-only EVM wallet monitor, in your terminal")

	intro := m.styles.Faint.Render(strings.Join([]string{
		"Track balances on Ethereum, Arbitrum, Base, Optimism and more,",
		"with transaction history and live gas.",
		"Read-only: never asks for private keys nor signs anything.",
	}, "\n"))

	commands := m.welcomeCommands()
	footer := m.welcomeFooter()

	// Probamos de la portada más rica a la más sobria y nos quedamos con la
	// primera que cabe (en alto y en ancho).
	build := func(withIntro, withCmds bool) string {
		parts := []string{logo, "", tagline}
		if withIntro {
			parts = append(parts, "", intro)
		}
		if withCmds {
			parts = append(parts, "", commands)
		}
		parts = append(parts, "", footer)
		return lipgloss.JoinVertical(lipgloss.Center, parts...)
	}

	introFits := useBig && lipgloss.Width(intro) <= m.contentW
	for _, cfg := range [][2]bool{{introFits, true}, {introFits, false}, {false, true}, {false, false}} {
		block := build(cfg[0], cfg[1])
		if lipgloss.Height(block) <= avail && lipgloss.Width(block) <= m.contentW {
			return block
		}
	}

	// Último recurso (terminal diminuto): recortamos para no desbordar.
	return lipgloss.NewStyle().MaxWidth(m.contentW).MaxHeight(avail).Render(build(false, false))
}

// welcomeCommands compone la línea de comandos básicos (tecla resaltada +
// acción), separados por puntos.
func (m Model) welcomeCommands() string {
	cmds := []struct{ key, desc string }{
		{"tab", "switch"},
		{"enter", "add"},
		{"r", "reload"},
		{"?", "shortcuts"},
	}
	parts := make([]string, len(cmds))
	for i, c := range cmds {
		parts[i] = m.styles.Brand.Render(c.key) + " " + m.styles.Faint.Render(c.desc)
	}
	return strings.Join(parts, m.styles.Faint.Render(" · "))
}

// welcomeFooter compone el pie de acciones de la portada, con el conmutador de
// "no volver a mostrar" reflejando su estado.
func (m Model) welcomeFooter() string {
	check := "[ ]"
	if m.welcomeHide {
		check = m.styles.Balance.Render("[x]")
	}
	return m.styles.TableHeader.Render("enter") + m.styles.Faint.Render(" start    ") +
		m.styles.Brand.Render("d") + " " + check + m.styles.Faint.Render(" don't show again    ") +
		m.styles.TableHeader.Render("q") + m.styles.Faint.Render(" quit")
}
