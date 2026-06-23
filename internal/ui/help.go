package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// keyHelp es un atajo y su descripción.
type keyHelp struct {
	key  string
	desc string
}

// keyGroup agrupa atajos por contexto.
type keyGroup struct {
	title string
	keys  []keyHelp
}

// keyGroups es la fuente de verdad de los atajos que muestra el overlay de ayuda.
func keyGroups() []keyGroup {
	return []keyGroup{
		{"Global", []keyHelp{
			{"tab / shift+tab", "switch tab"},
			{"ctrl+k", "command palette (search/go/actions)"},
			{"ctrl+g", "metrics / debug mode"},
			{"n", "choose visible networks"},
			{"?", "toggle this help"},
			{"q", "quit"},
			{"ctrl+c", "quit"},
		}},
		{"Accounts", []keyHelp{
			{"enter", "view wallet detail (or add if typing)"},
			{"ctrl+d", "delete selected wallet (confirm with ctrl+d)"},
			{"↑ ↓", "move through the list"},
		}},
		{"Balances", []keyHelp{
			{"↑ ↓", "move through the table"},
			{"f", "toggle all / selected wallet"},
			{"s / S", "sort column / reverse"},
			{"y", "copy address to clipboard"},
			{"o", "open in block explorer"},
			{"e", "export balances to CSV"},
			{"r", "reload balances"},
		}},
		{"Transactions", []keyHelp{
			{"↑ ↓", "move through the list"},
			{"enter", "view tx detail"},
			{"esc", "close detail"},
			{"f", "filter by network (cycle)"},
			{"s / S", "sort column / reverse"},
			{"m", "load more (next page)"},
			{"y", "copy hash to clipboard"},
			{"o", "open tx in explorer"},
			{"e", "export transactions to CSV"},
			{"r", "reload history"},
		}},
	}
}

// renderHelp dibuja el overlay de ayuda centrado en el área de contenido.
func (m Model) renderHelp() string {
	var b strings.Builder
	b.WriteString(m.styles.StateTitle.Render("⌨  Keyboard shortcuts"))
	b.WriteString("\n")

	for _, g := range keyGroups() {
		b.WriteString("\n")
		b.WriteString(m.styles.Balance.Render(g.title))
		b.WriteString("\n")
		for _, k := range g.keys {
			b.WriteString("  " + m.styles.Brand.Render(fit(k.key, 16)) + m.styles.Faint.Render(k.desc) + "\n")
		}
	}

	// Envolver en un panel con borde da un bloque rectangular de ancho fijo, así
	// las columnas quedan alineadas a la izquierda dentro del modal en vez de
	// centrarse línea a línea.
	panel := m.styles.Panel.Render(strings.TrimRight(b.String(), "\n"))
	return lipgloss.Place(m.contentW, m.contentH, lipgloss.Center, lipgloss.Center, panel)
}
