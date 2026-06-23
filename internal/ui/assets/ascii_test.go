package assets

import (
	"testing"

	"charm.land/lipgloss/v2"
)

// El banner debe ser un bloque rectangular: todas las líneas con el mismo ancho
// de celda. Si no, el logo sale escalonado en pantalla.
func TestLogoIsRectangular(t *testing.T) {
	lines := LogoLines()
	if len(lines) == 0 {
		t.Fatal("logo vacío")
	}
	w := lipgloss.Width(lines[0])
	for i, l := range lines {
		if got := lipgloss.Width(l); got != w {
			t.Errorf("línea %d ancho = %d, quiero %d (%q)", i, got, w, l)
		}
	}
	t.Logf("logo %d×%d", w, len(lines))
}
