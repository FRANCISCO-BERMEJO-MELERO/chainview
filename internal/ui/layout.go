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

	// La portada de bienvenida ocupa todo el frame (logo centrado), sin tabs ni
	// barra de estado: es una pantalla aparte hasta que el usuario entra.
	if m.showWelcome {
		welcome := lipgloss.Place(m.contentW, m.height-2, lipgloss.Center, lipgloss.Center, m.renderWelcome())
		return m.styles.Frame.Width(m.width).Render(welcome)
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

// renderStatusBar pinta el estado vivo (izq.) y la leyenda de atajos (der.). La
// leyenda se ajusta al ancho que deja libre la parte izquierda.
func (m Model) renderStatusBar(w int) string {
	left := m.statusLeft()
	avail := w - lipgloss.Width(left) - 1 // -1: hueco mínimo entre ambos
	return bar(left, m.statusRight(avail), w)
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
		return m.styles.Faint.Render("⟳ cargando…") + m.rateLimitedSuffix()
	}
	fresh := "⟳ " + sinceShort(m.lastGas)
	if m.gasTotal > 0 && m.gasOK < m.gasTotal {
		return m.styles.Faint.Render(fresh+" · ") +
			m.styles.NoticeError.Render(fmt.Sprintf("%d/%d redes con error", m.gasTotal-m.gasOK, m.gasTotal)) +
			m.rateLimitedSuffix()
	}
	return m.styles.Faint.Render(fmt.Sprintf("%s · %d/%d redes ok", fresh, m.gasOK, m.gasTotal)) +
		m.rateLimitedSuffix()
}

// rateLimitedSuffix devuelve un indicador persistente de redes en cooldown por
// rate-limit (2.9), o "" si no hay ninguna. Mientras dure el cooldown se ve en la
// status bar, no solo como toast efímero.
func (m Model) rateLimitedSuffix() string {
	if m.client == nil {
		return ""
	}
	if n := len(m.client.RateLimitedChains()); n > 0 {
		return m.styles.Faint.Render(" · ") +
			m.styles.NoticeError.Render(fmt.Sprintf("⚠ %d limitada(s)", n))
	}
	return ""
}

// statusRight compone la leyenda de atajos de la derecha. Cada atajo es un par
// "tecla acción"; se incluyen, por orden de importancia, los que quepan enteros en
// `avail` (nunca a medias), y siempre se mantiene el sufijo global (ayuda/salir).
// Los atajos que no quepan siguen documentados en el overlay de ayuda (?).
func (m Model) statusRight(avail int) string {
	if m.paletteOpen {
		return m.styles.Faint.Render("↑↓ elegir · enter ejecutar · esc cerrar")
	}
	if m.helpOpen {
		return m.styles.Faint.Render("? · esc cerrar")
	}
	if m.networksOpen {
		return m.styles.Faint.Render("espacio conmutar · esc cerrar")
	}

	// Sufijo global, siempre presente. En Cuentas se escribe en el input, así que
	// 'q' no sale (se sale con ctrl+c).
	suffix := "? ayuda · q salir"
	if m.active == tabAccounts {
		suffix = "? ayuda · ctrl+c salir"
	}

	const sep = " · "
	used := lipgloss.Width(suffix)
	parts := make([]string, 0)
	for _, it := range m.hintItems() {
		if used+lipgloss.Width(sep)+lipgloss.Width(it) > avail {
			break
		}
		parts = append(parts, it)
		used += lipgloss.Width(sep) + lipgloss.Width(it)
	}
	parts = append(parts, suffix)
	return m.styles.Faint.Render(strings.Join(parts, sep))
}

// hintItems son los atajos contextuales de la pestaña activa como pares "tecla
// acción", ordenados de más a menos importante (la leyenda incluye los que quepan).
func (m Model) hintItems() []string {
	switch m.active {
	case tabAccounts:
		return []string{"enter detalle", "ctrl+d borrar", "ctrl+k paleta", "↑↓ mover"}
	case tabBalances:
		return []string{"y copiar", "o abrir", "s ordenar", "f filtrar", "e exportar", "r recargar", "ctrl+k paleta", "n redes", "↑↓ mover"}
	case tabTransactions:
		if m.txDetailOpen {
			return []string{"↑↓ desplazar", "esc cerrar"}
		}
		return []string{"enter detalle", "y copiar", "o abrir", "s ordenar", "f filtrar red", "m cargar más", "e exportar", "r recargar", "ctrl+k paleta", "n redes", "↑↓ mover"}
	}
	return nil
}

// renderContent dibuja el cuerpo de la pestaña activa, recortado y rellenado al
// área de contenido exacta para que el frame mantenga su tamaño.
func (m Model) renderContent() string {
	if m.tourActive {
		return m.renderTour()
	}
	if m.paletteOpen {
		return m.renderPalette()
	}
	if m.helpOpen {
		return m.renderHelp()
	}
	if m.debugOpen {
		return m.renderDebug()
	}
	if m.networksOpen {
		return m.renderNetworks()
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
