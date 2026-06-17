// Package storage persiste los datos del usuario (las wallets seguidas) en JSON
// dentro del directorio de datos XDG. A diferencia de la config (TOML, estable),
// estos datos cambian a menudo, de ahí el formato y ubicación separados.
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/ethereum/go-ethereum/common"
)

// dataPath es la ruta relativa dentro del directorio de datos XDG
// (p.ej. ~/.local/share/chainview/wallets.json en Linux).
const dataPath = "chainview/wallets.json"

// Wallets gestiona la lista de direcciones seguidas y su persistencia en disco.
// No es thread-safe: se usa desde el hilo de Update de Bubble Tea.
type Wallets struct {
	path  string
	items []common.Address
}

// Load resuelve la ruta XDG y carga las wallets guardadas. Un archivo
// inexistente se trata como lista vacía (no es error).
func Load() (*Wallets, error) {
	path, err := xdg.DataFile(dataPath)
	if err != nil {
		return nil, fmt.Errorf("resolviendo ruta de datos: %w", err)
	}
	return loadFrom(path)
}

// loadFrom carga desde una ruta concreta (testeable sin entorno XDG real).
func loadFrom(path string) (*Wallets, error) {
	w := &Wallets{path: path}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return w, nil // sin archivo: lista vacía
	}
	if err != nil {
		return nil, fmt.Errorf("leyendo wallets: %w", err)
	}

	var raw []string
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parseando wallets %s: %w", path, err)
	}
	for _, s := range raw {
		// Ignoramos entradas inválidas en disco en vez de romper el arranque.
		if common.IsHexAddress(s) {
			w.items = append(w.items, common.HexToAddress(s))
		}
	}
	return w, nil
}

// List devuelve una copia de las direcciones seguidas (para que la UI no mute el
// estado interno por accidente).
func (w *Wallets) List() []common.Address {
	out := make([]common.Address, len(w.items))
	copy(out, w.items)
	return out
}

// Len devuelve cuántas wallets hay seguidas.
func (w *Wallets) Len() int { return len(w.items) }

// Add valida y añade una dirección, evita duplicados y persiste. Devuelve error
// si la dirección no es una address EVM válida.
func (w *Wallets) Add(s string) error {
	if !common.IsHexAddress(s) {
		return fmt.Errorf("dirección EVM inválida: %q", s)
	}
	addr := common.HexToAddress(s)
	for _, existing := range w.items {
		if existing == addr {
			return nil // ya está: idempotente
		}
	}
	w.items = append(w.items, addr)
	return w.save()
}

// Remove elimina una dirección (si existe) y persiste.
func (w *Wallets) Remove(addr common.Address) error {
	for i, existing := range w.items {
		if existing == addr {
			w.items = append(w.items[:i], w.items[i+1:]...)
			return w.save()
		}
	}
	return nil
}

// save escribe la lista a disco en JSON. Crea el directorio si hace falta y
// escribe de forma atómica (archivo temporal + rename) para no corromper el
// fichero si el proceso muere a media escritura.
func (w *Wallets) save() error {
	if err := os.MkdirAll(filepath.Dir(w.path), 0o755); err != nil {
		return fmt.Errorf("creando directorio de datos: %w", err)
	}

	// common.Address serializa a su string hex con checksum en JSON.
	data, err := json.MarshalIndent(w.items, "", "  ")
	if err != nil {
		return fmt.Errorf("serializando wallets: %w", err)
	}

	tmp := w.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("escribiendo wallets: %w", err)
	}
	if err := os.Rename(tmp, w.path); err != nil {
		return fmt.Errorf("renombrando wallets: %w", err)
	}
	return nil
}
