package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromMissingFileUsesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "no-existe.toml")

	cfg, err := loadFrom(path)
	if err != nil {
		t.Fatalf("archivo inexistente no debería dar error: %v", err)
	}
	if cfg.RefreshSeconds != defaultRefreshSeconds {
		t.Fatalf("RefreshSeconds = %d, quiero %d", cfg.RefreshSeconds, defaultRefreshSeconds)
	}
	if len(cfg.RPC) != 0 {
		t.Fatalf("sin archivo no debería haber overrides, hay %d", len(cfg.RPC))
	}
}

func TestLoadFromOverridesRPCAndRefresh(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
refresh_seconds = 30

[rpc]
ethereum = "https://eth.example.com"
base = "https://base.example.com"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadFrom(path)
	if err != nil {
		t.Fatalf("loadFrom: %v", err)
	}
	if cfg.RefreshSeconds != 30 {
		t.Fatalf("RefreshSeconds = %d, quiero 30", cfg.RefreshSeconds)
	}

	nets := cfg.Networks()
	got := map[string]string{}
	for _, n := range nets {
		got[n.Key] = n.RPCURL
	}
	if got["ethereum"] != "https://eth.example.com" {
		t.Fatalf("override ethereum no aplicado: %q", got["ethereum"])
	}
	if got["base"] != "https://base.example.com" {
		t.Fatalf("override base no aplicado: %q", got["base"])
	}
	// Una red sin override conserva su RPC público por defecto.
	if got["optimism"] == "" || got["optimism"] == "https://base.example.com" {
		t.Fatalf("optimism debería conservar su RPC por defecto, got %q", got["optimism"])
	}
}

func TestLoadFromInvalidTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("esto = no es = toml válido"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadFrom(path); err == nil {
		t.Fatal("esperaba error con TOML corrupto")
	}
}

func TestRefreshSecondsSanitized(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("refresh_seconds = 0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RefreshSeconds != defaultRefreshSeconds {
		t.Fatalf("refresh_seconds=0 debería saneamiento a %d, got %d", defaultRefreshSeconds, cfg.RefreshSeconds)
	}
}
