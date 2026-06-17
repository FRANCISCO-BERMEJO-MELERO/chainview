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

	networks := cfg.Networks()
	client := chain.NewClient(networks)
	defer client.Close()

	refresh := time.Duration(cfg.RefreshSeconds) * time.Second
	m := ui.NewModel(client, wallets, networks, refresh)
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return err
	}
	return nil
}
