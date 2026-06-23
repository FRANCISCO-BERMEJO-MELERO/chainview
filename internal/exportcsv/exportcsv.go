// Package exportcsv escribe listas (cabecera + filas) a ficheros CSV. No conoce
// la UI ni los tipos de dominio: recibe ya las filas como [][]string, de modo que
// se puede testear de forma aislada y reutilizar para cualquier exportación.
package exportcsv

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Dir es la carpeta de exportaciones por defecto, relativa al directorio de
// trabajo actual. Predecible y fácil de encontrar para el usuario.
const Dir = "chainview-exports"

// Write crea `dir` si no existe y escribe un CSV nuevo `dir/name` con la cabecera
// y las filas dadas. Devuelve la ruta escrita.
func Write(dir, name string, header []string, rows [][]string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creando carpeta de exportación: %w", err)
	}
	path := filepath.Join(dir, name)

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("creando %s: %w", path, err)
	}
	defer func() { _ = f.Close() }() // red de seguridad ante returns tempranos

	w := csv.NewWriter(f)
	records := make([][]string, 0, len(rows)+1)
	records = append(records, header)
	records = append(records, rows...)
	if err := w.WriteAll(records); err != nil { // WriteAll hace Flush internamente
		return "", fmt.Errorf("escribiendo CSV: %w", err)
	}
	// Cerramos explícitamente para no perder un error de escritura en el flush final.
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("cerrando %s: %w", path, err)
	}
	return path, nil
}

// Timestamp da la marca de tiempo para los nombres de fichero (orden lexicográfico
// = orden cronológico).
func Timestamp(t time.Time) string {
	return t.Format("20060102-150405")
}

// SanitizeName deja un texto apto para nombre de fichero: solo letras, dígitos y
// . - _ ; cualquier otro carácter pasa a '-'. Vacío cae a "wallet".
func SanitizeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := b.String()
	if out == "" {
		return "wallet"
	}
	return out
}
