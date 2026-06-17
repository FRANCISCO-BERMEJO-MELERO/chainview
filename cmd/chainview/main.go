package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/ui"
)

func main() {
	m := ui.NewModel()
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
