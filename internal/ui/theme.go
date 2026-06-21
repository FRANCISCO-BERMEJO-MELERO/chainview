package ui

import "charm.land/lipgloss/v2"

// Paleta de chainview: violeta como color de marca + verde menta como acento.
// Se definen como constantes para tener un único punto de verdad de los colores;
// ningún View debería usar códigos de color sueltos: todo pasa por Styles.
const (
	colorViolet = "#8B5CF6" // marca / título
	colorMint   = "#5EEAD4" // acento / tab activo
	colorText   = "#E5E7EB" // texto principal
	colorFaint  = "#6B7280" // texto tenue (ayudas, secundario)
	colorError  = "#F87171" // errores
	colorGreen  = "#34D399" // tendencia a la baja (gas más barato)
	colorBorder = "#3F3F5A" // bordes de paneles
)

// Styles agrupa todos los estilos reutilizables de la TUI. Se construye una vez
// (DefaultStyles) y se guarda en el Model, de modo que las vistas solo consultan
// estilos ya definidos en lugar de crearlos ad hoc.
type Styles struct {
	Title       lipgloss.Style
	TabActive   lipgloss.Style
	TabInactive lipgloss.Style
	Panel       lipgloss.Style
	Error       lipgloss.Style
	Faint       lipgloss.Style
	Spinner     lipgloss.Style
	Balance     lipgloss.Style
	TrendUp     lipgloss.Style // gas sube (rojo)
	TrendDown   lipgloss.Style // gas baja (verde)

	Frame       lipgloss.Style // borde de la app que llena el terminal
	Brand       lipgloss.Style // "CHAINVIEW" en el header
	Rule        lipgloss.Style // separadores horizontales del frame
	StateTitle  lipgloss.Style // título de un estado vacío/error centrado
	NoticeError lipgloss.Style // toast/aviso de error en el footer
	NoticeInfo  lipgloss.Style // toast/aviso informativo en el footer
}

// DefaultStyles devuelve el tema por defecto (violeta + verde menta).
func DefaultStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorViolet)).
			Bold(true),

		TabActive: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMint)).
			Bold(true).
			Padding(0, 2),

		TabInactive: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorFaint)).
			Padding(0, 2),

		Panel: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText)).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorder)).
			Padding(1, 2),

		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorError)).
			Bold(true),

		Faint: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorFaint)),

		Spinner: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMint)),

		Balance: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMint)).
			Bold(true),

		TrendUp: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorError)).
			Bold(true),

		TrendDown: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorGreen)).
			Bold(true),

		Frame: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorBorder)).
			Padding(0, 1),

		Brand: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorViolet)).
			Bold(true),

		Rule: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorBorder)),

		StateTitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorText)).
			Bold(true),

		NoticeError: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorError)).
			Bold(true),

		NoticeInfo: lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMint)),
	}
}
