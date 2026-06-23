package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Motor de tablas a mano de chainview. Las tablas se construyen repartiendo el
// ancho de contenido disponible entre columnas con un mínimo garantizado y
// columnas "flex" que absorben (o ceden) el espacio sobrante, de modo que la
// tabla siempre llena el ancho y los números se alinean a la derecha.

const (
	tableGutter = 2 // ancho del marcador de selección ("› " / "  ")
	tableColGap = 1 // separación entre columnas contiguas
)

// colAlign indica cómo se ancla el contenido de una columna a su ancho.
type colAlign int

const (
	alignLeft colAlign = iota
	alignRight
)

// column describe una columna: título, alineación, ancho mínimo y si es flexible
// (reparte el espacio sobrante de la tabla).
type column struct {
	title string
	align colAlign
	min   int
	flex  bool
}

// tcell es una celda ya resuelta: su texto plano (para medir y alinear) y el
// estilo a aplicar cuando la fila no está seleccionada.
type tcell struct {
	text  string
	style lipgloss.Style
}

// txt construye una celda sin estilo propio (hereda el color de texto base).
func txt(s string) tcell { return tcell{text: s, style: lipgloss.NewStyle()} }

// styledCell construye una celda con un estilo concreto (color de símbolo, error…).
func styledCell(s string, st lipgloss.Style) tcell { return tcell{text: s, style: st} }

// layoutColumns reparte `total` celdas entre las columnas: cada una recibe al
// menos su mínimo y las flex absorben el sobrante a partes iguales. Si no cabe,
// recorta primero de las flex y luego del resto sin bajar ninguna de 1.
func layoutColumns(cols []column, total int) []int {
	n := len(cols)
	widths := make([]int, n)
	gaps := 0
	if n > 1 {
		gaps = (n - 1) * tableColGap
	}
	avail := total - tableGutter - gaps

	used, flexes := 0, 0
	for i, c := range cols {
		widths[i] = c.min
		used += c.min
		if c.flex {
			flexes++
		}
	}

	switch extra := avail - used; {
	case extra > 0 && flexes > 0:
		per, rem := extra/flexes, extra%flexes
		for i := range cols {
			if cols[i].flex {
				widths[i] += per
				if rem > 0 {
					widths[i]++
					rem--
				}
			}
		}
	case extra > 0:
		// Sin columnas flex: el sobrante engorda la última columna.
		widths[n-1] += extra
	case extra < 0:
		shrinkColumns(cols, widths, -extra)
	}
	return widths
}

// shrinkColumns descuenta `deficit` celdas, primero de las columnas flex y luego
// del resto, dando vueltas hasta resolverlo o dejar todas en su mínimo de 1.
func shrinkColumns(cols []column, widths []int, deficit int) {
	for _, flexOnly := range []bool{true, false} {
		for deficit > 0 {
			shrunk := false
			for i := range cols {
				if deficit == 0 {
					break
				}
				if flexOnly && !cols[i].flex {
					continue
				}
				if widths[i] > 1 {
					widths[i]--
					deficit--
					shrunk = true
				}
			}
			if !shrunk {
				break
			}
		}
	}
}

// alignCell ajusta un texto a un ancho exacto según su alineación, truncando con
// … si sobra.
func alignCell(s string, w int, a colAlign) string {
	if a == alignRight {
		return fitRight(s, w)
	}
	return fit(s, w)
}

// tableWidth es el ancho total que ocupa la tabla (marcador + columnas + gaps).
func tableWidth(widths []int) int {
	w := tableGutter
	for i, cw := range widths {
		w += cw
		if i > 0 {
			w += tableColGap
		}
	}
	return w
}

// tableHeader compone la fila de cabecera: el gutter en blanco (para alinear con
// el marcador de las filas) y los títulos con estilo de cabecera.
func (m Model) tableHeader(cols []column, widths []int) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = alignCell(c.title, widths[i], c.align)
	}
	body := strings.Join(parts, strings.Repeat(" ", tableColGap))
	return strings.Repeat(" ", tableGutter) + m.styles.TableHeader.Render(body)
}

// tableRule devuelve una regla del ancho exacto de la tabla, para separar la
// cabecera de los datos.
func (m Model) tableRule(widths []int) string {
	return m.styles.Rule.Render(strings.Repeat("─", tableWidth(widths)))
}

// tableRow compone una fila de datos. Si está seleccionada, la pinta como una
// barra resaltada a ancho completo (texto uniforme, sin estilos por celda) para
// que el realce sea legible aun sin color; si no, aplica el estilo propio de cada
// celda y antepone dos espacios donde iría el marcador.
func (m Model) tableRow(cols []column, widths []int, cells []tcell, selected bool) string {
	gap := strings.Repeat(" ", tableColGap)
	parts := make([]string, len(cells))

	if selected {
		for i, c := range cells {
			parts[i] = alignCell(c.text, widths[i], cols[i].align)
		}
		return m.styles.RowSelected.Render("› " + strings.Join(parts, gap))
	}

	for i, c := range cells {
		parts[i] = c.style.Render(alignCell(c.text, widths[i], cols[i].align))
	}
	return "  " + strings.Join(parts, gap)
}

// fitRight ajusta un string a un ancho fijo alineándolo a la derecha: rellena con
// espacios por la izquierda si falta, o trunca con … si sobra. Es el helper para
// columnas numéricas, donde alinear la derecha permite comparar magnitudes.
func fitRight(s string, w int) string {
	r := []rune(s)
	if len(r) > w {
		if w <= 1 {
			return string(r[:w])
		}
		return string(r[:w-1]) + "…"
	}
	return strings.Repeat(" ", w-len(r)) + s
}
