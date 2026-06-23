package ui

import (
	"testing"

	"charm.land/lipgloss/v2"
)

// La portada nunca debe desbordar el frame: a cualquier tamaño soportado, el
// frame con la bienvenida sigue midiendo exactamente lo que el terminal.
func TestWelcomeFillsTerminalExactly(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}, {60, 18}} {
		m := testModel(size[0], size[1])
		m.showWelcome = true
		out := m.renderFrame()
		if gotW := lipgloss.Width(out); gotW != size[0] {
			t.Errorf("welcome %dx%d: ancho = %d, quiero %d", size[0], size[1], gotW, size[0])
		}
		if gotH := lipgloss.Height(out); gotH != size[1] {
			t.Errorf("welcome %dx%d: alto = %d, quiero %d", size[0], size[1], gotH, size[1])
		}
	}
}

// El bloque interior de la portada cabe en el alto disponible (no se recorta).
func TestWelcomeBlockFitsHeight(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}, {60, 18}} {
		m := testModel(size[0], size[1])
		m.showWelcome = true
		block := m.renderWelcome()
		if h := lipgloss.Height(block); h > size[1]-2 {
			t.Errorf("welcome %dx%d: bloque alto %d > disponible %d", size[0], size[1], h, size[1]-2)
		}
		if w := lipgloss.Width(block); w > m.contentW {
			t.Errorf("welcome %dx%d: bloque ancho %d > contentW %d", size[0], size[1], w, m.contentW)
		}
	}
}
