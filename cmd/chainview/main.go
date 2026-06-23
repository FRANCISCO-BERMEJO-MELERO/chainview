package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/config"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/storage"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/ui"
)

// Información de build, inyectable con -ldflags "-X main.version=...". Los nombres
// coinciden con los que rellena goreleaser por defecto, así que no hace falta
// configurar plantillas de ldflags a mano. En un `go install` sin ldflags se
// recae en runtime/debug.ReadBuildInfo (ver versionString).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	fs := flag.NewFlagSet("chainview", flag.ExitOnError)
	showVersion := fs.Bool("version", false, "muestra la versión y sale")
	debugFlag := fs.Bool("debug", false, "arranca con el overlay de métricas/debug visible")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "chainview — monitor de wallets EVM en el terminal (watch-only).\n\n")
		fmt.Fprintf(os.Stderr, "Uso:\n  chainview [opciones]\n\nOpciones:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEntorno:\n  CHAINVIEW_DEBUG=1   equivale a --debug\n  ETHERSCAN_API_KEY   usa Etherscan para el historial (opcional)\n")
	}
	// flag.ExitOnError ya gestiona -h/--help (imprime Usage y sale con código 0).
	_ = fs.Parse(os.Args[1:])

	if *showVersion {
		fmt.Println(versionString())
		return
	}

	// CHAINVIEW_DEBUG=1 tiene la misma fuerza que --debug (3.3).
	debugMode := *debugFlag || os.Getenv("CHAINVIEW_DEBUG") == "1"

	if err := run(debugMode); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// versionString compone la línea de versión. Si el binario se construyó con
// ldflags (goreleaser o `make build`) usa esos valores; si no (p. ej. `go
// install`), recae en la información de build del módulo.
func versionString() string {
	v, c, d := version, commit, date
	if v == "dev" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
				v = bi.Main.Version
			}
			for _, s := range bi.Settings {
				switch s.Key {
				case "vcs.revision":
					if c == "none" {
						c = s.Value
					}
				case "vcs.time":
					if d == "unknown" {
						d = s.Value
					}
				}
			}
		}
	}
	return fmt.Sprintf("chainview %s (commit %s, %s)", v, c, d)
}

// run monta las dependencias y arranca la TUI. Se separa de main para que los
// defer (cerrar el cliente) se ejecuten siempre.
func run(debugMode bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	wallets, err := storage.Load()
	if err != nil {
		return err
	}

	prefs, err := storage.LoadPrefs()
	if err != nil {
		return err
	}

	networks := cfg.Networks()
	client := chain.NewClient(networks)
	defer client.Close()

	// Historial de txs: por defecto usamos Blockscout, que es gratis y SIN API
	// key (la app funciona recién clonada, sin que nadie tenga que registrarse).
	// Solo si el usuario configura una key de Etherscan usamos Etherscan. La
	// variable de entorno tiene prioridad sobre el TOML para no obligar a dejar la
	// key en un archivo.
	apiKey := cfg.EtherscanAPIKey
	if env := os.Getenv("ETHERSCAN_API_KEY"); env != "" {
		apiKey = env
	}
	blockscout := chain.NewBlockscoutProvider(networks)
	var txProvider chain.TxProvider = blockscout
	if apiKey != "" {
		txProvider = chain.NewEtherscanProvider(apiKey)
	}

	// Resolver ENS (solo mainnet) sobre la conexión del cliente; con caché propia.
	ens := chain.NewENSResolver(client)

	// Precios fiat keyless (DefiLlama) para valorar saldos y total de cartera.
	prices := chain.NewDefiLlamaPrices(networks)

	refresh := time.Duration(cfg.RefreshSeconds) * time.Second
	// El descubrimiento de tokens (1.2) usa siempre Blockscout (keyless), aunque el
	// historial de txs vaya por Etherscan.
	m := ui.NewModel(client, wallets, networks, refresh, txProvider, ens, prices, cfg.FiatCurrency, blockscout, cfg.Theme, prefs).
		WithDebug(debugMode)
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return err
	}
	return nil
}
