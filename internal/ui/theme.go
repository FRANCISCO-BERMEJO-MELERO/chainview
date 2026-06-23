package ui

import (
	"charm.land/lipgloss/v2"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
)

// Palette son los colores de un tema, por rol (no por nombre de color). Es el
// único punto de verdad de los colores: ninguna vista usa códigos sueltos, todo
// pasa por Styles, que se construye a partir de una Palette (StylesFor). Así
// añadir un tema es solo definir otra Palette.
type Palette struct {
	Violet string // marca / título
	Mint   string // acento / tab activo / saldos
	Text   string // texto principal
	Faint  string // texto tenue (ayudas, secundario)
	Error  string // errores
	Green  string // tendencia a la baja (gas más barato) / ok
	Border string // bordes de paneles
	SelBg  string // fondo de la fila seleccionada
	SelFg  string // texto sobre la fila seleccionada
	Amber  string // tx saliente (tag OUT)

	// Colores de marca de cada red, para el badge de la columna "Red".
	Eth  string
	Arb  string
	Base string
	Op   string
}

// paletteDark es el tema por defecto (violeta + verde menta sobre fondo oscuro).
var paletteDark = Palette{
	Violet: "#8B5CF6",
	Mint:   "#5EEAD4",
	Text:   "#E5E7EB",
	Faint:  "#6B7280",
	Error:  "#F87171",
	Green:  "#34D399",
	Border: "#3F3F5A",
	SelBg:  "#332B57",
	SelFg:  "#F5F3FF",
	Amber:  "#FBBF24",
	Eth:    "#7C8CF0",
	Arb:    "#28A0F0",
	Base:   "#3C7DFF",
	Op:     "#FF5C5C",
}

// paletteLight es el tema claro: mismos roles, colores legibles sobre fondo
// claro (texto oscuro, acentos más saturados, selección violeta muy tenue).
var paletteLight = Palette{
	Violet: "#6D28D9",
	Mint:   "#0D9488",
	Text:   "#1F2937",
	Faint:  "#6B7280",
	Error:  "#DC2626",
	Green:  "#047857",
	Border: "#C7C7D9",
	SelBg:  "#EDE9FE",
	SelFg:  "#4C1D95",
	Amber:  "#B45309",
	Eth:    "#4F5FD0",
	Arb:    "#1577C9",
	Base:   "#2563EB",
	Op:     "#DC2626",
}

// themeNames son los presets válidos para la config (además de "auto").
const (
	themeDark  = "dark"
	themeLight = "light"
	themeAuto  = "auto"
)

// paletteByName resuelve una paleta concreta (dark/light). "auto" y cualquier
// valor desconocido caen a oscuro: es el comportamiento histórico y el más
// seguro hasta que la detección del fondo (BackgroundColorMsg) diga otra cosa.
func paletteByName(name string) Palette {
	if name == themeLight {
		return paletteLight
	}
	return paletteDark
}

// Styles agrupa todos los estilos reutilizables de la TUI. Se construye una vez
// (StylesFor) y se guarda en el Model, de modo que las vistas solo consultan
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

	TableHeader lipgloss.Style // cabecera de columnas de una tabla
	RowSelected lipgloss.Style // barra de la fila seleccionada en una tabla
	Symbol      lipgloss.Style // símbolo de moneda / dato secundario en una tabla
	Ok          lipgloss.Style // estado correcto (✓ verde)

	// Tags de tipo de tx (entrante/saliente/…) y badges de red.
	TagIn   lipgloss.Style
	TagOut  lipgloss.Style
	TagSelf lipgloss.Style
	TagCall lipgloss.Style
	TagNew  lipgloss.Style
	Badges  map[uint64]lipgloss.Style // chain ID -> estilo del badge de red
}

// DefaultStyles devuelve el tema oscuro por defecto. Atajo de StylesFor(paletteDark)
// que conservan los tests y el arranque hasta resolver el tema configurado.
func DefaultStyles() Styles {
	return StylesFor(paletteDark)
}

// StylesFor construye el conjunto de estilos a partir de una paleta.
func StylesFor(p Palette) Styles {
	col := lipgloss.Color
	return Styles{
		Title: lipgloss.NewStyle().
			Foreground(col(p.Violet)).
			Bold(true),

		TabActive: lipgloss.NewStyle().
			Foreground(col(p.Mint)).
			Bold(true).
			Padding(0, 2),

		TabInactive: lipgloss.NewStyle().
			Foreground(col(p.Faint)).
			Padding(0, 2),

		Panel: lipgloss.NewStyle().
			Foreground(col(p.Text)).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(col(p.Border)).
			Padding(1, 2),

		Error: lipgloss.NewStyle().
			Foreground(col(p.Error)).
			Bold(true),

		Faint: lipgloss.NewStyle().
			Foreground(col(p.Faint)),

		Spinner: lipgloss.NewStyle().
			Foreground(col(p.Mint)),

		Balance: lipgloss.NewStyle().
			Foreground(col(p.Mint)).
			Bold(true),

		TrendUp: lipgloss.NewStyle().
			Foreground(col(p.Error)).
			Bold(true),

		TrendDown: lipgloss.NewStyle().
			Foreground(col(p.Green)).
			Bold(true),

		Frame: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(col(p.Border)).
			Padding(0, 1),

		Brand: lipgloss.NewStyle().
			Foreground(col(p.Violet)).
			Bold(true),

		Rule: lipgloss.NewStyle().
			Foreground(col(p.Border)),

		StateTitle: lipgloss.NewStyle().
			Foreground(col(p.Text)).
			Bold(true),

		NoticeError: lipgloss.NewStyle().
			Foreground(col(p.Error)).
			Bold(true),

		NoticeInfo: lipgloss.NewStyle().
			Foreground(col(p.Mint)),

		// Cabecera de tabla: texto pleno en negrita para que estructure la tabla
		// (a diferencia del faint de los datos secundarios).
		TableHeader: lipgloss.NewStyle().
			Foreground(col(p.Text)).
			Bold(true),

		// Fila seleccionada: barra de fondo a ancho completo. El realce no depende
		// solo del color (el bloque se ve aunque el terminal no tenga truecolor), y
		// se combina con el marcador "›".
		RowSelected: lipgloss.NewStyle().
			Background(col(p.SelBg)).
			Foreground(col(p.SelFg)).
			Bold(true),

		Symbol: lipgloss.NewStyle().
			Foreground(col(p.Faint)),

		Ok: lipgloss.NewStyle().
			Foreground(col(p.Green)),

		// Tags de tipo: el color refuerza el significado (verde recibe, ámbar
		// envía), pero el texto/símbolo siempre acompaña (no solo color).
		TagIn:   lipgloss.NewStyle().Foreground(col(p.Green)).Bold(true),
		TagOut:  lipgloss.NewStyle().Foreground(col(p.Amber)).Bold(true),
		TagSelf: lipgloss.NewStyle().Foreground(col(p.Faint)),
		TagCall: lipgloss.NewStyle().Foreground(col(p.Mint)),
		TagNew:  lipgloss.NewStyle().Foreground(col(p.Violet)).Bold(true),

		Badges: map[uint64]lipgloss.Style{
			chain.ChainEthereum: lipgloss.NewStyle().Foreground(col(p.Eth)).Bold(true),
			chain.ChainArbitrum: lipgloss.NewStyle().Foreground(col(p.Arb)).Bold(true),
			chain.ChainBase:     lipgloss.NewStyle().Foreground(col(p.Base)).Bold(true),
			chain.ChainOptimism: lipgloss.NewStyle().Foreground(col(p.Op)).Bold(true),
		},
	}
}
