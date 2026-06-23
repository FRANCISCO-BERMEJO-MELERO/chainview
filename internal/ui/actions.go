package ui

import (
	"os/exec"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/atotto/clipboard"
	"github.com/ethereum/go-ethereum/common"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
)

// copiedMsg informa del resultado de copiar al portapapeles (2.4).
type copiedMsg struct {
	label string
	err   error
}

// openedMsg informa del resultado de abrir una URL en el navegador (2.5).
type openedMsg struct{ err error }

// copyToClipboardCmd copia un valor al portapapeles en background. Aísla la
// dependencia atotto/clipboard para poder probar el resto sin portapapeles real.
func copyToClipboardCmd(value, label string) tea.Cmd {
	return func() tea.Msg {
		return copiedMsg{label: label, err: clipboard.WriteAll(value)}
	}
}

// openURLCmd abre una URL en el navegador del sistema en background.
func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg { return openedMsg{err: openURL(url)} }
}

// openURL abre una URL con el comando del SO (sin dependencias nuevas): xdg-open
// en Linux/BSD, open en macOS, el handler de protocolo en Windows.
func openURL(url string) error {
	var name string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		name, args = "open", []string{url}
	case "windows":
		name, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		name, args = "xdg-open", []string{url}
	}
	return exec.Command(name, args...).Start()
}

// networkByID busca una red por chain ID: primero en las activas, luego en el
// catálogo completo (por si la fila es de una red recién desactivada).
func (m Model) networkByID(id uint64) (chain.Network, bool) {
	for _, n := range m.networks {
		if n.ChainID == id {
			return n, true
		}
	}
	for _, n := range m.allNetworks {
		if n.ChainID == id {
			return n, true
		}
	}
	return chain.Network{}, false
}

// explorerAddressURL construye la URL de una address en el explorador de su red,
// o false si la red no tiene explorador configurado.
func (m Model) explorerAddressURL(chainID uint64, addr common.Address) (string, bool) {
	n, ok := m.networkByID(chainID)
	if !ok || n.Explorer == "" {
		return "", false
	}
	return strings.TrimRight(n.Explorer, "/") + "/address/" + addr.Hex(), true
}

// explorerTxURL construye la URL de una transacción en el explorador de su red.
func (m Model) explorerTxURL(chainID uint64, hash string) (string, bool) {
	n, ok := m.networkByID(chainID)
	if !ok || n.Explorer == "" {
		return "", false
	}
	return strings.TrimRight(n.Explorer, "/") + "/tx/" + hash, true
}
