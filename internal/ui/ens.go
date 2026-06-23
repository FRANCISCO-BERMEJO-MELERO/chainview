package ui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/ethereum/go-ethereum/common"
)

// ensResolvedMsg lleva los nombres ENS resueltos para un conjunto de direcciones.
type ensResolvedMsg struct {
	names map[common.Address]string
}

// ensAddMsg es el resultado de resolver un nombre ENS escrito en el input de
// Cuentas para añadirlo como wallet.
type ensAddMsg struct {
	name string
	addr common.Address
	ok   bool
}

// resolveAndAddCmd resuelve un nombre ENS (directa) en background para añadirlo
// como wallet. El que llama ya ha comprobado que m.ens no es nil.
func (m Model) resolveAndAddCmd(name string) tea.Cmd {
	resolver := m.ens
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		addr, ok := resolver.Resolve(ctx, name)
		return ensAddMsg{name: name, addr: addr, ok: ok}
	}
}

// resolveWalletNamesCmd resuelve en background el nombre ENS inverso de todas las
// wallets seguidas. El propio ENSResolver cachea, así que llamarlo de más es
// barato. Devuelve nil si no hay resolver (p.ej. no se pudo abrir mainnet).
func (m Model) resolveWalletNamesCmd() tea.Cmd {
	if m.ens == nil {
		return nil
	}
	resolver := m.ens
	addrs := m.wallets.List()
	if len(addrs) == 0 {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		names := make(map[common.Address]string, len(addrs))
		for _, a := range addrs {
			if name, ok := resolver.Lookup(ctx, a); ok {
				names[a] = name
			}
		}
		return ensResolvedMsg{names: names}
	}
}

// displayName devuelve el nombre ENS de una dirección si lo conocemos, o su forma
// corta 0x1234…abcd en caso contrario. Es el punto único para mostrar una
// dirección de forma legible en cualquier pestaña.
func (m Model) displayName(a common.Address) string {
	if name := m.ensNames[a]; name != "" {
		return name
	}
	return shortAddr(a)
}
