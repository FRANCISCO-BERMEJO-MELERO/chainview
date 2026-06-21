package exportcsv

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteCreatesDirAndCSV(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "exports") // no existe aún
	header := []string{"a", "b"}
	rows := [][]string{
		{"1", "texto, con coma"}, // fuerza quoting
		{"2", `con "comillas"`},  // fuerza escapado
	}

	path, err := Write(dir, "test.csv", header, rows)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if filepath.Dir(path) != dir {
		t.Errorf("ruta = %s, quiero dentro de %s", path, dir)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("abrir CSV: %v", err)
	}
	defer f.Close()
	recs, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("leyendo CSV (¿quoting roto?): %v", err)
	}
	if len(recs) != 3 {
		t.Fatalf("esperaba 3 registros (cabecera+2), hay %d", len(recs))
	}
	if recs[0][0] != "a" || recs[1][1] != "texto, con coma" || recs[2][1] != `con "comillas"` {
		t.Errorf("contenido inesperado: %v", recs)
	}
}

func TestTimestampFormat(t *testing.T) {
	got := Timestamp(time.Date(2026, 6, 21, 15, 30, 5, 0, time.UTC))
	if got != "20260621-153005" {
		t.Errorf("Timestamp = %q", got)
	}
}

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"vitalik.eth": "vitalik.eth",
		"0x1234…abcd": "0x1234-abcd", // el … no es ASCII -> '-'
		"":            "wallet",
		"a/b\\c:d":    "a-b-c-d",
	}
	for in, want := range cases {
		if got := SanitizeName(in); got != want {
			t.Errorf("SanitizeName(%q) = %q, quiero %q", in, got, want)
		}
	}
}
