package ui

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// update regenera los ficheros golden en vez de compararlos: `go test -update`.
var update = flag.Bool("update", false, "regenera los ficheros golden de la TUI")

// plain quita los códigos ANSI de un frame para que los golden sean texto plano,
// deterministas en cualquier entorno (sin depender del color profile del terminal).
func plain(s string) string {
	return ansi.Strip(s)
}

// assertGolden compara un frame (ya sin ANSI) contra testdata/golden/<name>.golden.
// Con -update lo (re)escribe; sin él falla mostrando ambas versiones si difieren.
func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name+".golden")
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir golden: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("escribir golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("leer golden %s: %v (¿falta -update?)", path, err)
	}
	if got != string(want) {
		t.Errorf("frame %q no coincide con el golden.\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

// TestGoldenFrames fija el render de estados representativos de la TUI. Los modelos
// son deterministas (sin tiempo ni red). Regenerar con: go test ./internal/ui -update
func TestGoldenFrames(t *testing.T) {
	cases := []struct {
		name  string
		setup func() Model
	}{
		{"balances-sin-wallets", func() Model { return testModel(80, 24) }},
		{"cuentas", func() Model {
			m := testModel(80, 24)
			m.active = tabAccounts
			return m
		}},
		{"transacciones-sin-wallets", func() Model {
			m := testModel(80, 24)
			m.active = tabTransactions
			return m
		}},
		{"overlay-ayuda", func() Model {
			m := testModel(80, 24)
			m.helpOpen = true
			return m
		}},
		{"overlay-debug", func() Model {
			m := testModel(80, 24)
			m.debugOpen = true
			return m
		}},
		{"degradado-pequeno", func() Model { return testModel(40, 10) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := plain(tc.setup().renderFrame())
			assertGolden(t, tc.name, got)
		})
	}
}
