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

func TestNetworkBlockAddsAndOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
# Sobreescribe solo el RPC de Ethereum (override parcial por chain_id).
[[network]]
chain_id = 1
rpc_url  = "https://eth.custom.example"

# Da de alta una red nueva (Gnosis).
[[network]]
key            = "gnosis"
name           = "Gnosis"
chain_id       = 100
rpc_url        = "https://gnosis-rpc.example"
symbol         = "xDAI"
blockscout_api = "https://gnosis.blockscout.com/api"

# Inválida: sin chain_id, debe ignorarse.
[[network]]
name    = "Fantasma"
rpc_url = "https://nope.example"

# Inválida: red nueva sin rpc_url, debe ignorarse.
[[network]]
chain_id = 999999
name     = "SinRPC"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadFrom(path)
	if err != nil {
		t.Fatalf("loadFrom: %v", err)
	}
	nets := cfg.Networks()

	byID := map[uint64]struct {
		key, rpc, symbol, blockscout string
	}{}
	for _, n := range nets {
		byID[n.ChainID] = struct{ key, rpc, symbol, blockscout string }{n.Key, n.RPCURL, n.Symbol, n.BlockscoutAPI}
	}

	// Override parcial: cambia el RPC pero conserva metadatos por defecto.
	eth := byID[1]
	if eth.rpc != "https://eth.custom.example" {
		t.Errorf("RPC de Ethereum no sobreescrito: %q", eth.rpc)
	}
	if eth.key != "ethereum" || eth.blockscout == "" {
		t.Errorf("override parcial perdió metadatos: %+v", eth)
	}

	// Alta de red nueva.
	gnosis, ok := byID[100]
	if !ok {
		t.Fatal("la red nueva (chain_id 100) no se añadió")
	}
	if gnosis.symbol != "xDAI" || gnosis.rpc != "https://gnosis-rpc.example" {
		t.Errorf("red nueva mal construida: %+v", gnosis)
	}

	// Las inválidas no entran.
	if _, ok := byID[999999]; ok {
		t.Error("una red nueva sin rpc_url no debería añadirse")
	}
}

func TestFiatCurrencyDefaultAndSanitize(t *testing.T) {
	dir := t.TempDir()

	// Sin archivo: default usd.
	cfg, _ := loadFrom(filepath.Join(dir, "missing.toml"))
	if cfg.FiatCurrency != "usd" {
		t.Fatalf("default FiatCurrency = %q, quiero usd", cfg.FiatCurrency)
	}

	// Mayúsculas se normalizan a minúsculas.
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(`fiat_currency = "USD"`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FiatCurrency != "usd" {
		t.Fatalf("FiatCurrency = %q, quiero usd (normalizado)", cfg.FiatCurrency)
	}
}

func TestThemeDefaultAndSanitize(t *testing.T) {
	dir := t.TempDir()

	// Sin archivo: default auto.
	cfg, _ := loadFrom(filepath.Join(dir, "missing.toml"))
	if cfg.Theme != "auto" {
		t.Fatalf("default Theme = %q, quiero auto", cfg.Theme)
	}

	// Valor válido en mayúsculas se normaliza; valor desconocido cae a auto.
	for in, want := range map[string]string{"LIGHT": "light", "Dark": "dark", "rosa": "auto"} {
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte(`theme = "`+in+`"`+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := loadFrom(path)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Theme != want {
			t.Errorf("theme=%q -> %q, quiero %q", in, cfg.Theme, want)
		}
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
