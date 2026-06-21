package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// Tamaño mínimo de terminal para dibujar el frame. Por debajo degradamos a un
// aviso en vez de romper el layout.
const (
	minWidth  = 60
	minHeight = 18
)

// frameChrome son las líneas que NO son contenido dentro del frame: 2 del borde
// + header + 3 reglas + tab bar + status bar.
const frameChrome = 8

// contentDims calcula el área útil de contenido (ancho × alto) a partir del
// tamaño de terminal. El ancho descuenta borde (2) y padding horizontal (2); el
// alto descuenta frameChrome.
func contentDims(width, height int) (w, h int) {
	w = width - 4
	h = height - frameChrome
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

// renderFrame compone toda la pantalla: header + tabs + contenido + status bar
// dentro de un borde que llena el terminal. Es el único punto que dibuja el
// chrome de la app.
func (m Model) renderFrame() string {
	switch {
	case m.width == 0 || m.height == 0:
		return "iniciando…"
	case m.width < minWidth || m.height < minHeight:
		msg := m.styles.Brand.Render("chainview") + "\n\n" +
			m.styles.Faint.Render("Amplía la terminal\n(mín. 60×18)")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
	}

	iw := m.contentW
	rule := m.styles.Rule.Render(strings.Repeat("─", iw))

	inner := lipgloss.JoinVertical(lipgloss.Left,
		m.renderHeaderBar(iw),
		rule,
		lipgloss.PlaceHorizontal(iw, lipgloss.Left, m.renderTabs()),
		rule,
		m.renderContent(),
		rule,
		m.renderStatusBar(iw),
	)
	// Frame.Width es el ancho TOTAL (border-box en lipgloss v2): el borde y el
	// padding se descuentan de aquí, dejando justo `iw` para las secciones.
	return m.styles.Frame.Width(m.width).Render(inner)
}

// renderHeaderBar pinta la marca a la izquierda y el resumen de gas a la derecha.
func (m Model) renderHeaderBar(w int) string {
	return bar(m.styles.Brand.Render("CHAINVIEW"), m.renderGasHeader(), w)
}

// renderStatusBar pinta el estado/atajos contextuales (izq.) y los atajos
// globales (der.). El contenido vivo del estado se enriquece en pasos posteriores.
func (m Model) renderStatusBar(w int) string {
	return bar(m.statusLeft(), m.statusRight(), w)
}

// statusLeft muestra el toast activo o, si no hay, el estado vivo: frescura del
// último refresco y salud de las redes.
func (m Model) statusLeft() string {
	if m.notice != "" && time.Now().Before(m.noticeUntil) {
		style := m.styles.NoticeInfo
		if m.noticeLevel == noticeError {
			style = m.styles.NoticeError
		}
		return style.Render(m.notice)
	}
	if m.lastGas.IsZero() {
		return m.styles.Faint.Render("⟳ cargando…")
	}
	fresh := "⟳ " + sinceShort(m.lastGas)
	if m.gasTotal > 0 && m.gasOK < m.gasTotal {
		return m.styles.Faint.Render(fresh+" · ") +
			m.styles.NoticeError.Render(fmt.Sprintf("%d/%d redes con error", m.gasTotal-m.gasOK, m.gasTotal))
	}
	return m.styles.Faint.Render(fmt.Sprintf("%s · %d/%d redes ok", fresh, m.gasOK, m.gasTotal))
}

// statusRight muestra los atajos contextuales + el acceso a la ayuda.
func (m Model) statusRight() string {
	if m.helpOpen {
		return m.styles.Faint.Render("? / esc cerrar")
	}
	return m.styles.Faint.Render(m.contextHint() + " · ? ayuda · q salir")
}

// contextHint son los atajos relevantes de la pestaña activa, en versión corta
// para el footer (la lista completa vive en el overlay de ayuda).
func (m Model) contextHint() string {
	switch m.active {
	case tabAccounts:
		return "enter añadir · ctrl+d borrar · ↑↓ mover"
	case tabBalances:
		return "↑↓ mover · r recargar"
	case tabTransactions:
		if m.txDetailOpen {
			return "↑↓ desplazar · esc cerrar"
		}
		return "↑↓ · enter detalle · r recargar"
	}
	return ""
}

// renderContent dibuja el cuerpo de la pestaña activa, recortado y rellenado al
// área de contenido exacta para que el frame mantenga su tamaño.
func (m Model) renderContent() string {
	if m.helpOpen {
		return m.renderHelp()
	}

	var body string
	switch m.active {
	case tabAccounts:
		body = m.renderAccounts()
	case tabBalances:
		body = m.renderBalances()
	case tabTransactions:
		body = m.renderTransactions()
	}

	// MaxWidth/MaxHeight recortan cualquier desbordamiento (tablas anchas o muchas
	// filas); Place rellena hasta el alto exacto, alineado arriba-izquierda.
	body = lipgloss.NewStyle().MaxWidth(m.contentW).MaxHeight(m.contentH).Render(body)
	return lipgloss.Place(m.contentW, m.contentH, lipgloss.Left, lipgloss.Top, body)
}

// renderState dibuja un estado (vacío/carga/error) centrado en el área de
// contenido, con estilo uniforme para las tres pestañas.
func (m Model) renderState(icon, title, hint string) string {
	body := m.styles.StateTitle.Render(icon + "  " + title)
	if hint != "" {
		body += "\n\n" + m.styles.Faint.Render(hint)
	}
	return lipgloss.Place(m.contentW, m.contentH, lipgloss.Center, lipgloss.Center, body)
}

// sinceShort da una marca temporal relativa fina (con segundos) para el estado
// del footer.
func sinceShort(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < 2*time.Second:
		return "ahora"
	case d < time.Minute:
		return fmt.Sprintf("hace %ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("hace %dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("hace %dh", int(d.Hours()))
	}
}

// bar coloca `left` y `right` en una línea de ancho w, con el espacio repartido
// entre ambos. Si no caben, recorta de forma segura (sin desbordar el frame).
func bar(left, right string, w int) string {
	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return lipgloss.NewStyle().MaxWidth(w).Render(left + " " + right)
	}
	return left + strings.Repeat(" ", gap) + right
}
