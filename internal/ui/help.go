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
			{"tab / shift+tab", "cambiar de pestaña"},
			{"?", "abrir/cerrar esta ayuda"},
			{"q", "salir"},
			{"ctrl+c", "salir"},
		}},
		{"Cuentas", []keyHelp{
			{"enter", "añadir wallet (address o nombre ENS)"},
			{"ctrl+d", "borrar la wallet seleccionada"},
			{"↑ ↓", "moverse por la lista"},
		}},
		{"Balances", []keyHelp{
			{"↑ ↓", "moverse por la tabla"},
			{"r", "recargar balances"},
		}},
		{"Transacciones", []keyHelp{
			{"↑ ↓", "moverse por la lista"},
			{"enter", "ver el detalle de la tx"},
			{"esc", "cerrar el detalle"},
			{"r", "recargar historial"},
		}},
	}
}

// renderHelp dibuja el overlay de ayuda centrado en el área de contenido.
func (m Model) renderHelp() string {
	var b strings.Builder
	b.WriteString(m.styles.StateTitle.Render("⌨  Atajos de teclado"))
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
