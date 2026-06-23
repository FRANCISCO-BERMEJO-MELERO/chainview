package ui

import (
	"testing"

	"charm.land/lipgloss/v2"
)

// TestDebugOverlayTogglesAndFits comprueba que ctrl+g abre/cierra el overlay de
// debug (3.3) y que su frame llena el terminal exacto.
func TestDebugOverlayTogglesAndFits(t *testing.T) {
	m := testModel(80, 24)

	r, _ := m.Update(keyMsg("ctrl+g"))
	m = r.(Model)
	if !m.debugOpen {
		t.Fatal("ctrl+g debería abrir el overlay de debug")
	}

	out := m.renderFrame()
	if w, h := lipgloss.Width(out), lipgloss.Height(out); w != 80 || h != 24 {
		t.Errorf("frame del overlay = %dx%d, quiero 80x24", w, h)
	}

	r, _ = m.Update(keyMsg("esc"))
	if r.(Model).debugOpen {
		t.Error("esc debería cerrar el overlay de debug")
	}
}

// TestWithDebugStartsOpen verifica que --debug (WithDebug) arranca con el overlay
// visible.
func TestWithDebugStartsOpen(t *testing.T) {
	if !testModel(80, 24).WithDebug(true).debugOpen {
		t.Error("WithDebug(true) debería dejar debugOpen=true")
	}
}
