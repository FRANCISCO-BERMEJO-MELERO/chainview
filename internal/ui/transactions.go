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

// txRow es una tx con su descripción legible ya calculada. La descripción se
// resuelve en background (decodifica el calldata y consulta metadatos de token),
// no en el render, que debe ser rápido y sin I/O.
type txRow struct {
	tx     chain.Tx
	detail string
}

// txsLoadedMsg / txsErrMsg son los resultados de cargar el historial de una
// wallet. Llevan la dirección para descartar respuestas de una wallet que ya no
// es la seleccionada (evita pintar datos obsoletos al cambiar rápido).
type txsLoadedMsg struct {
	wallet common.Address
	rows   []txRow
}

type txsErrMsg struct {
	wallet common.Address
	err    error
}

// fetchTxsCmd carga el historial en background con timeout y, de paso, decodifica
// cada tx a una descripción legible (resolviendo símbolos/decimales de token).
func (m Model) fetchTxsCmd(wallet common.Address) tea.Cmd {
	provider := m.txProvider
	resolver := m.client
	chainID := m.txChainID
	nativeSymbol := m.networkSymbol(chainID)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		txs, err := provider.RecentTxs(ctx, chainID, wallet, txLimit)
		if err != nil {
			return txsErrMsg{wallet: wallet, err: err}
		}

		rows := make([]txRow, len(txs))
		for i, tx := range txs {
			rows[i] = txRow{
				tx:     tx,
				detail: describeTx(ctx, resolver, chainID, wallet, nativeSymbol, tx),
			}
		}
		return txsLoadedMsg{wallet: wallet, rows: rows}
	}
}

// describeTx produce la descripción legible de una tx para la columna "Detalle":
// una acción de token decodificada cuando es posible, una transferencia nativa
// con su valor, o un fallback para calls desconocidas.
func describeTx(ctx context.Context, r chain.TokenResolver, chainID uint64, wallet common.Address, nativeSymbol string, tx chain.Tx) string {
	if call, ok := chain.DecodeCall(tx.Input); ok {
		return describeTokenCall(ctx, r, chainID, tx.To, call)
	}

	// Sin calldata decodable: o es una transferencia nativa de ETH...
	if tx.Value.Sign() > 0 {
		dir := "→ " + shortAddr(tx.To)
		if tx.From != wallet {
			dir = "← " + shortAddr(tx.From)
		}
		return dir + "  " + chain.FormatEther(tx.Value) + " " + nativeSymbol
	}
	// ...o una llamada a contrato que no sabemos leer (mostramos el selector).
	if len(tx.Input) >= 4 {
		return "call 0x" + common.Bytes2Hex(tx.Input[:4])
	}
	return "—"
}

// describeTokenCall formatea una llamada de token ya decodificada. Si conocemos
// los metadatos del token aplica decimales y símbolo ("100 USDC"); si no, cae a
// la cantidad cruda para no romper ante un token desconocido.
func describeTokenCall(ctx context.Context, r chain.TokenResolver, chainID uint64, token common.Address, call chain.DecodedCall) string {
	meta, known := r.TokenMeta(ctx, chainID, token)
	amount := func() string {
		if known {
			return chain.FormatUnits(call.Value, int(meta.Decimals)) + " " + meta.Symbol
		}
		return call.Value.String()
	}

	switch call.Kind {
	case chain.CallTransfer, chain.CallTransferFrom:
		return "Transfer " + amount() + " → " + shortAddr(call.To)
	case chain.CallApprove:
		sym := "token"
		if known {
			sym = meta.Symbol
		}
		return "Approve " + sym + " → " + shortAddr(call.To)
	default:
		return ""
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
	// Con el modal de detalle abierto, esc lo cierra y el resto de teclas
	// (flechas, page up/down) las consume el viewport para desplazar.
	if m.txDetailOpen {
		if msg.String() == "esc" {
			m.txDetailOpen = false
			return m, nil
		}
		var cmd tea.Cmd
		m.txViewport, cmd = m.txViewport.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "up":
		if m.txCursor > 0 {
			m.txCursor--
		}
	case "down":
		if m.txCursor < len(m.txs)-1 {
			m.txCursor++
		}
	case "enter":
		if m.txCursor >= 0 && m.txCursor < len(m.txs) {
			m.txViewport.SetContent(m.txDetailContent(m.txs[m.txCursor]))
			m.txViewport.GotoTop()
			m.txDetailOpen = true
		}
	case "r":
		if m.txState != stateLoading {
			m.txState = stateIdle // forzar recarga
			return m, m.loadTxsCmd()
		}
	}
	return m, nil
}

// txDetailContent compone el cuerpo del modal con todos los campos de la tx.
func (m Model) txDetailContent(row txRow) string {
	tx := row.tx
	sym := m.networkSymbol(m.txChainID)

	status := "ok"
	if !tx.Success {
		status = "fallida"
	}

	label := func(s string) string { return m.styles.Faint.Render(fit(s, 12)) }
	addr := func(a common.Address) string {
		if name := m.ensNames[a]; name != "" {
			return name + "  " + a.Hex()
		}
		return a.Hex()
	}

	lines := []string{
		label("Hash") + tx.Hash,
		label("Estado") + status,
		label("Bloque") + fmt.Sprintf("%d", tx.BlockNumber),
		label("Fecha") + tx.Timestamp.Format("2006-01-02 15:04:05") + "  (" + humanizeSince(tx.Timestamp) + ")",
		"",
		label("De") + addr(tx.From),
		label("A") + addr(tx.To),
		label("Valor") + chain.FormatEther(tx.Value) + " " + sym,
		label("Acción") + row.detail,
		"",
		label("Gas usado") + fmt.Sprintf("%d", tx.GasUsed),
		label("Gas price") + chain.FormatUnits(tx.GasPrice, 9) + " gwei",
		label("Nonce") + fmt.Sprintf("%d", tx.Nonce),
	}
	return strings.Join(lines, "\n")
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
	if m.txDetailOpen {
		return m.txViewport.View()
	}

	if m.wallets.Len() == 0 {
		return m.styles.Faint.Render("Añade wallets en la pestaña Cuentas para ver sus transacciones.")
	}

	header := m.styles.Faint.Render("Wallet "+m.displayName(m.txWallet)+" · "+m.networkName(m.txChainID)) + "\n\n"

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
	b.WriteString(m.styles.Faint.Render(fit("Tx", 12) + fit("Detalle", 42) + fit("Cuándo", 11) + "Estado"))
	b.WriteString("\n")

	for i, row := range m.txs {
		hash := fit(row.tx.Hash[:10]+"…", 12)
		detail := fit(row.detail, 42)
		when := fit(humanizeSince(row.tx.Timestamp), 11)

		status := "ok"
		if !row.tx.Success {
			status = "fallida"
		}

		switch {
		case i == m.txCursor:
			b.WriteString(m.styles.Balance.Render("› " + hash + detail + when + status))
		case !row.tx.Success:
			b.WriteString("  " + hash + detail + when + m.styles.Error.Render(status))
		default:
			b.WriteString("  " + hash + detail + when + status)
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
