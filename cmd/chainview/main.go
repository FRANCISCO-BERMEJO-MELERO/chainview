package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/ui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// run monta las dependencias y arranca la TUI. Se separa de main para que el
// defer client.Close() se ejecute siempre (os.Exit en main lo saltaría).
func run() error {
	client := chain.NewClient(chain.DefaultNetworks())
	defer client.Close()

	m := ui.NewModel(client)
	if _, err := tea.NewProgram(m).Run(); err != nil {
		return err
	}
	return nil
}
