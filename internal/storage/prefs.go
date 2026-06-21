package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

// prefsPath es la ruta relativa de las preferencias dentro del directorio de
// datos XDG (p.ej. ~/.local/share/chainview/prefs.json en Linux).
const prefsPath = "chainview/prefs.json"

// Prefs guarda las preferencias del usuario que la TUI persiste entre sesiones.
// A diferencia de la config (TOML, que edita el usuario a mano), estas las cambia
// la propia app, de ahí el JSON separado. No es thread-safe: se usa desde el hilo
// de Update de Bubble Tea.
type Prefs struct {
	path string

	// HideWelcome indica que el usuario pidió no volver a ver la pantalla de
	// bienvenida al arrancar.
	HideWelcome bool `json:"hide_welcome"`
}

// LoadPrefs resuelve la ruta XDG y carga las preferencias guardadas. Un archivo
// inexistente se trata como preferencias por defecto (no es error).
func LoadPrefs() (*Prefs, error) {
	path, err := xdg.DataFile(prefsPath)
	if err != nil {
		return nil, fmt.Errorf("resolviendo ruta de preferencias: %w", err)
	}
	return loadPrefsFrom(path)
}

// loadPrefsFrom carga desde una ruta concreta (testeable sin entorno XDG real).
func loadPrefsFrom(path string) (*Prefs, error) {
	p := &Prefs{path: path}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return p, nil // sin archivo: valores por defecto
	}
	if err != nil {
		return nil, fmt.Errorf("leyendo preferencias: %w", err)
	}
	if err := json.Unmarshal(data, p); err != nil {
		return nil, fmt.Errorf("parseando preferencias %s: %w", path, err)
	}
	return p, nil
}

// SetHideWelcome fija la preferencia de ocultar la bienvenida y la persiste.
func (p *Prefs) SetHideWelcome(v bool) error {
	p.HideWelcome = v
	return p.save()
}

// save escribe las preferencias a disco en JSON, de forma atómica (archivo
// temporal + rename) y creando el directorio si hace falta.
func (p *Prefs) save() error {
	if p.path == "" {
		return nil // Prefs sin ruta (p.ej. en tests): no persiste
	}
	if err := os.MkdirAll(filepath.Dir(p.path), 0o755); err != nil {
		return fmt.Errorf("creando directorio de datos: %w", err)
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("serializando preferencias: %w", err)
	}

	tmp := p.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("escribiendo preferencias: %w", err)
	}
	if err := os.Rename(tmp, p.path); err != nil {
		return fmt.Errorf("renombrando preferencias: %w", err)
	}
	return nil
}
