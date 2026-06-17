package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/ethereum/go-ethereum/common"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
)

// txLimit es cuántas transacciones recientes pedimos por wallet.
const txLimit = 20

// txsLoadedMsg / txsErrMsg son los resultados de cargar el historial de una
// wallet. Llevan la dirección para descartar respuestas de una wallet que ya no
// es la seleccionada (evita pintar datos obsoletos al cambiar rápido).
type txsLoadedMsg struct {
	wallet common.Address
	txs    []chain.Tx
}

type txsErrMsg struct {
	wallet common.Address
	err    error
}

// fetchTxsCmd carga el historial en background con timeout.
func (m Model) fetchTxsCmd(wallet common.Address) tea.Cmd {
	provider := m.txProvider
	chainID := m.txChainID
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		txs, err := provider.RecentTxs(ctx, chainID, wallet, txLimit)
		if err != nil {
			return txsErrMsg{wallet: wallet, err: err}
		}
		return txsLoadedMsg{wallet: wallet, txs: txs}
	}
}

// selectedWallet devuelve la wallet sobre la que actúa la pestaña Transacciones:
// la seleccionada en Cuentas. El segundo valor es false si no hay wallets.
func (m Model) selectedWallet() (common.Address, bool) {
	addrs := m.wallets.List()
	if len(addrs) == 0 {
		return common.Address{}, false
	}
	idx := m.accCursor
	if idx < 0 || idx >= len(addrs) {
		idx = 0
	}
	return addrs[idx], true
}

// loadTxsCmd arranca la carga del historial para la wallet seleccionada si hace
// falta (wallet distinta o aún sin cargar). Receptor por puntero: muta el estado.
func (m *Model) loadTxsCmd() tea.Cmd {
	wallet, ok := m.selectedWallet()
	if !ok {
		return nil
	}
	if m.txState != stateIdle && m.txWallet == wallet {
		return nil // ya cargada (o cargando) para esta wallet
	}
	m.txWallet = wallet
	m.txCursor = 0
	m.txState = stateLoading
	return tea.Batch(m.spinner.Tick, m.fetchTxsCmd(wallet))
}

// updateTransactions maneja las teclas de la pestaña Transacciones.
func (m Model) updateTransactions(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		if m.txCursor > 0 {
			m.txCursor--
		}
	case "down":
		if m.txCursor < len(m.txs)-1 {
			m.txCursor++
		}
	case "r":
		if m.txState != stateLoading {
			m.txState = stateIdle // forzar recarga
			return m, m.loadTxsCmd()
		}
	}
	return m, nil
}

func (m *Model) clampTxCursor() {
	if m.txCursor >= len(m.txs) {
		m.txCursor = len(m.txs) - 1
	}
	if m.txCursor < 0 {
		m.txCursor = 0
	}
}

func (m Model) renderTransactions() string {
	if m.wallets.Len() == 0 {
		return m.styles.Faint.Render("Añade wallets en la pestaña Cuentas para ver sus transacciones.")
	}

	header := m.styles.Faint.Render("Wallet "+shortAddr(m.txWallet)+" · "+m.networkName(m.txChainID)) + "\n\n"

	switch {
	case m.txState == stateLoading && len(m.txs) == 0:
		return header + m.spinner.View() + " cargando transacciones…"
	case m.txState == stateError:
		return header + m.styles.Error.Render("error: "+m.txErr.Error())
	case m.txState == stateLoaded && len(m.txs) == 0:
		return header + m.styles.Faint.Render("Sin transacciones en "+m.networkName(m.txChainID)+".")
	}

	var b strings.Builder
	b.WriteString(header)
	if m.txState == stateLoading {
		b.WriteString(m.styles.Faint.Render(m.spinner.View()+" actualizando…") + "\n")
	}
	b.WriteString(m.styles.Faint.Render(fit("Tx", 12) + fit("Contraparte", 16) + fit("Valor", 14) + fit("Cuándo", 11) + "Estado"))
	b.WriteString("\n")

	for i, tx := range m.txs {
		hash := fit(tx.Hash[:10]+"…", 12)

		var counterparty string
		if tx.From == m.txWallet {
			counterparty = "→ " + shortAddr(tx.To)
		} else {
			counterparty = "← " + shortAddr(tx.From)
		}
		counterparty = fit(counterparty, 16)

		value := fit(chain.FormatEther(tx.Value)+" ETH", 14)
		when := fit(humanizeSince(tx.Timestamp), 11)

		status := "ok"
		if !tx.Success {
			status = "fallida"
		}

		switch {
		case i == m.txCursor:
			b.WriteString(m.styles.Balance.Render("› " + hash + counterparty + value + when + status))
		case !tx.Success:
			b.WriteString("  " + hash + counterparty + value + when + m.styles.Error.Render(status))
		default:
			b.WriteString("  " + hash + counterparty + value + when + status)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// humanizeSince da una marca temporal relativa y compacta (hace 3h, hace 2d).
func humanizeSince(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "ahora"
	case d < time.Hour:
		return fmt.Sprintf("hace %dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("hace %dh", int(d.Hours()))
	default:
		return fmt.Sprintf("hace %dd", int(d.Hours()/24))
	}
}
