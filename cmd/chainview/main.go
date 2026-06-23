package main

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/config"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/storage"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// debugEnabled indica si arrancar con el overlay de métricas/debug visible (3.3),
// vía la flag --debug o la variable de entorno CHAINVIEW_DEBUG=1.
func debugEnabled() bool {
	if os.Getenv("CHAINVIEW_DEBUG") == "1" {
		return true
	}
	for _, a := range os.Args[1:] {
		if a == "--debug" || a == "-debug" {
			return true
		}
	}
	return false
}

// run monta las dependencias y arranca la TUI. Se separa de main para que los
// defer (cerrar el cliente) se ejecuten siempre.
func run() error {
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
		WithDebug(debugEnabled())
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return err
	}
	return nil
}
