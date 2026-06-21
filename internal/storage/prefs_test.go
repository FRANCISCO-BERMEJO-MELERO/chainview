package storage

import (
	"path/filepath"
	"testing"
)

func TestPrefsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prefs.json")

	// Sin archivo: valores por defecto.
	p, err := loadPrefsFrom(path)
	if err != nil {
		t.Fatalf("loadPrefsFrom inicial: %v", err)
	}
	if p.HideWelcome {
		t.Error("HideWelcome debería ser false por defecto")
	}

	// Guardar y releer desde disco.
	if err := p.SetHideWelcome(true); err != nil {
		t.Fatalf("SetHideWelcome: %v", err)
	}
	reloaded, err := loadPrefsFrom(path)
	if err != nil {
		t.Fatalf("loadPrefsFrom tras guardar: %v", err)
	}
	if !reloaded.HideWelcome {
		t.Error("HideWelcome no se persistió")
	}
}

// Unas preferencias sin ruta (las de un test) no deben fallar al guardar.
func TestPrefsSaveWithoutPathNoop(t *testing.T) {
	p := &Prefs{}
	if err := p.SetHideWelcome(true); err != nil {
		t.Errorf("guardar sin ruta debería ser no-op, got %v", err)
	}
}
