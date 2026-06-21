package ui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ethereum/go-ethereum/common"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
)

// txPageSize es cuántas transacciones pedimos por red en cada página.
const txPageSize = 50

// txRow es una tx con su descripción legible ya calculada. La descripción se
// resuelve en background (decodifica el calldata y consulta metadatos de token),
// no en el render, que debe ser rápido y sin I/O.
type txRow struct {
	tx     chain.Tx
	detail string
}

// txReq pide una página concreta de una red.
type txReq struct {
	chainID uint64
	page    int
}

// txNetResult es el resultado de pedir una página a una red (error aislado: el
// fallo de una red no invalida las demás).
type txNetResult struct {
	chainID uint64
	page    int
	rows    []txRow
	err     error
}

// txPageMsg transporta el resultado de una tanda de páginas (una o varias redes).
// Lleva la wallet para descartar respuestas de una wallet que ya no es la
// seleccionada (evita pintar datos obsoletos al cambiar rápido).
type txPageMsg struct {
	wallet  common.Address
	results []txNetResult
}

// activeTxChains son las redes activas sobre las que cargamos txs, respetando el
// filtro de red (txNetFilter: 0 = todas).
func (m Model) activeTxChains() []uint64 {
	out := make([]uint64, 0, len(m.networks))
	for _, n := range m.networks {
		if m.txNetFilter != 0 && n.ChainID != m.txNetFilter {
			continue
		}
		out = append(out, n.ChainID)
	}
	return out
}

// nextTxFilter cicla el filtro de red: todas → primera red → … → última → todas.
func (m Model) nextTxFilter() uint64 {
	ids := make([]uint64, 0, len(m.networks)+1)
	ids = append(ids, 0) // "todas"
	for _, n := range m.networks {
		ids = append(ids, n.ChainID)
	}
	for i, id := range ids {
		if id == m.txNetFilter {
			return ids[(i+1)%len(ids)]
		}
	}
	return 0
}

// visibleTxs son las txs que se muestran según el filtro de red.
func (m Model) visibleTxs() []txRow {
	if m.txNetFilter == 0 {
		return m.txs
	}
	out := make([]txRow, 0, len(m.txs))
	for _, r := range m.txs {
		if r.tx.ChainID == m.txNetFilter {
			out = append(out, r)
		}
	}
	return out
}

// fetchTxPagesCmd pide en paralelo las páginas indicadas (una goroutine por red),
// decodificando de paso cada tx a su descripción legible. Error por red aislado.
func (m Model) fetchTxPagesCmd(wallet common.Address, reqs []txReq) tea.Cmd {
	provider := m.txProvider
	resolver := m.client
	symbols := make(map[uint64]string, len(m.allNetworks))
	for _, n := range m.allNetworks {
		symbols[n.ChainID] = n.Symbol
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		results := make([]txNetResult, len(reqs))
		var wg sync.WaitGroup
		for i, req := range reqs {
			i, req := i, req
			wg.Add(1)
			go func() {
				defer wg.Done()
				txs, err := provider.RecentTxs(ctx, req.chainID, wallet, req.page, txPageSize)
				if err != nil {
					results[i] = txNetResult{chainID: req.chainID, page: req.page, err: err}
					return
				}
				rows := make([]txRow, len(txs))
				for j, tx := range txs {
					rows[j] = txRow{
						tx:     tx,
						detail: describeTx(ctx, resolver, req.chainID, wallet, symbols[req.chainID], tx),
					}
				}
				results[i] = txNetResult{chainID: req.chainID, page: req.page, rows: rows}
			}()
		}
		wg.Wait()
		return txPageMsg{wallet: wallet, results: results}
	}
}

// sortDedupTxRows ordena por fecha desc y elimina duplicados por (red, hash).
func sortDedupTxRows(rows []txRow) []txRow {
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].tx.Timestamp.After(rows[j].tx.Timestamp)
	})
	seen := make(map[string]bool, len(rows))
	out := rows[:0]
	for _, r := range rows {
		key := strconv.FormatUint(r.tx.ChainID, 10) + ":" + r.tx.Hash
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, r)
	}
	return out
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

// loadTxsCmd arranca (o reinicia) la carga del historial multi-red para la wallet
// seleccionada: resetea la paginación y pide la primera página de cada red activa.
// Receptor por puntero: muta el estado.
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
	m.txScroll = 0
	m.txs = nil
	m.txPage = map[uint64]int{}
	m.txExhausted = map[uint64]bool{}
	m.txState = stateLoading

	reqs := make([]txReq, 0)
	for _, id := range m.activeTxChains() {
		reqs = append(reqs, txReq{chainID: id, page: 1})
	}
	if len(reqs) == 0 {
		m.txState = stateLoaded
		return nil
	}
	return tea.Batch(m.spinner.Tick, m.fetchTxPagesCmd(wallet, reqs))
}

// loadMoreTxsCmd pide la siguiente página de cada red activa no agotada y la
// fusiona con lo ya cargado. No hace nada si ya hay una carga en curso o si no
// queda nada por traer.
func (m *Model) loadMoreTxsCmd() tea.Cmd {
	if m.txState == stateLoading {
		return nil
	}
	reqs := make([]txReq, 0)
	for _, id := range m.activeTxChains() {
		if m.txExhausted[id] {
			continue
		}
		reqs = append(reqs, txReq{chainID: id, page: m.txPage[id] + 1})
	}
	if len(reqs) == 0 {
		return nil // nada más que cargar
	}
	m.txState = stateLoading
	return tea.Batch(m.spinner.Tick, m.fetchTxPagesCmd(m.txWallet, reqs))
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
		m.txScroll = clampScroll(m.txCursor, len(m.visibleTxs()), m.txListCapacity(), m.txScroll)
	case "down":
		vis := m.visibleTxs()
		if m.txCursor < len(vis)-1 {
			m.txCursor++
		}
		m.txScroll = clampScroll(m.txCursor, len(vis), m.txListCapacity(), m.txScroll)
		// Al llegar al final de lo cargado, intentamos traer más automáticamente.
		if m.txCursor == len(vis)-1 {
			if cmd := m.loadMoreTxsCmd(); cmd != nil {
				return m, cmd
			}
		}
	case "m":
		if cmd := m.loadMoreTxsCmd(); cmd != nil {
			return m, cmd
		}
	case "f":
		m.txNetFilter = m.nextTxFilter()
		m.txCursor = 0
		m.txScroll = 0
	case "e":
		return m, m.exportTxsCmd()
	case "enter":
		vis := m.visibleTxs()
		if m.txCursor >= 0 && m.txCursor < len(vis) {
			m.txViewport.SetContent(m.txDetailContent(vis[m.txCursor]))
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

// txTag devuelve la etiqueta visible y el estilo de un tipo de tx.
func (m Model) txTag(k chain.TxKind) (string, lipgloss.Style) {
	switch k {
	case chain.KindIn:
		return "↓ IN", m.styles.TagIn
	case chain.KindOut:
		return "↑ OUT", m.styles.TagOut
	case chain.KindSelf:
		return "⇄ SELF", m.styles.TagSelf
	case chain.KindCall:
		return "⚙ CALL", m.styles.TagCall
	case chain.KindNew:
		return "✦ NEW", m.styles.TagNew
	default:
		return "·", m.styles.Faint
	}
}

// badgeStyle es el estilo de color del badge de una red (fallback tenue).
func (m Model) badgeStyle(chainID uint64) lipgloss.Style {
	if s, ok := m.styles.Badges[chainID]; ok {
		return s
	}
	return m.styles.Faint
}

// networkBadge es la etiqueta corta de red ya coloreada (para el detalle).
func (m Model) networkBadge(chainID uint64) string {
	return m.badgeStyle(chainID).Render(gasLabel(chainID))
}

// txDetailContent compone el cuerpo del modal con todos los campos de la tx,
// incluyendo los más técnicos (red, tipo, selector del método).
func (m Model) txDetailContent(row txRow) string {
	tx := row.tx
	sym := m.networkSymbol(tx.ChainID)
	kind := chain.ClassifyTx(tx, m.txWallet)
	tagText, _ := m.txTag(kind)

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
		label("Red") + m.networkBadge(tx.ChainID) + m.styles.Faint.Render(fmt.Sprintf("  %s (chain %d)", m.networkName(tx.ChainID), tx.ChainID)),
		label("Tipo") + tagText,
		label("Hash") + tx.Hash,
		label("Estado") + status,
		label("Bloque") + fmt.Sprintf("%d", tx.BlockNumber),
		label("Fecha") + tx.Timestamp.Format("2006-01-02 15:04:05") + "  (" + humanizeSince(tx.Timestamp) + ")",
		"",
		label("De") + addr(tx.From),
		label("A") + addr(tx.To),
		label("Valor") + chain.FormatEther(tx.Value) + " " + sym,
		label("Acción") + row.detail,
	}
	if len(tx.Input) >= 4 {
		lines = append(lines, label("Selector")+"0x"+common.Bytes2Hex(tx.Input[:4]))
	}
	lines = append(lines,
		"",
		label("Gas usado")+fmt.Sprintf("%d", tx.GasUsed),
		label("Gas price")+chain.FormatUnits(tx.GasPrice, 9)+" gwei",
		label("Nonce")+fmt.Sprintf("%d", tx.Nonce),
	)
	return strings.Join(lines, "\n")
}

// clampTxCursor mantiene el cursor dentro de la lista visible.
func (m *Model) clampTxCursor() {
	n := len(m.visibleTxs())
	if m.txCursor >= n {
		m.txCursor = n - 1
	}
	if m.txCursor < 0 {
		m.txCursor = 0
	}
	m.txScroll = clampScroll(m.txCursor, n, m.txListCapacity(), m.txScroll)
}

// txListCapacity es cuántas filas de tx caben en el área de contenido, descontando
// el cromo de la pestaña (contexto + cabecera + regla + pie).
func (m Model) txListCapacity() int {
	rows := m.contentH - 5
	if rows < 1 {
		rows = 1
	}
	return rows
}

// clampScroll ajusta el desplazamiento para que el cursor quede siempre visible.
func clampScroll(cursor, total, capacity, scroll int) int {
	if capacity < 1 {
		capacity = 1
	}
	if cursor < scroll {
		scroll = cursor
	}
	if cursor >= scroll+capacity {
		scroll = cursor - capacity + 1
	}
	if scroll > total-capacity {
		scroll = total - capacity
	}
	if scroll < 0 {
		scroll = 0
	}
	return scroll
}

func (m Model) renderTransactions() string {
	if m.txDetailOpen {
		return m.txViewport.View()
	}

	if m.wallets.Len() == 0 {
		return m.renderState("◯", "Sin wallets", "Añádelas en la pestaña Cuentas para ver sus transacciones")
	}

	switch {
	case m.txState == stateLoading && len(m.txs) == 0:
		return m.renderState(m.spinner.View(), "Cargando transacciones…", "")
	case m.txState == stateError:
		return m.renderState("⚠", "No se pudo cargar el historial", m.txErr.Error()+" — pulsa r para reintentar")
	case m.txState == stateLoaded && len(m.txs) == 0:
		return m.renderState("◯", "Sin transacciones", "para "+m.displayName(m.txWallet))
	}

	vis := m.visibleTxs()
	cols := txColumns()
	widths := layoutColumns(cols, m.contentW)

	// Contexto: wallet en primer plano + filtro de red + nº de txs (+ spinner si
	// estamos cargando más).
	filterLabel := "todas las redes"
	if m.txNetFilter != 0 {
		filterLabel = m.networkName(m.txNetFilter)
	}
	ctx := m.styles.Balance.Render(m.displayName(m.txWallet)) +
		m.styles.Faint.Render(" · "+filterLabel+" · "+fmt.Sprintf("%d txs", len(vis)))
	if m.txState == stateLoading {
		ctx += m.styles.Faint.Render("  " + m.spinner.View())
	}

	var b strings.Builder
	b.WriteString(ctx + "\n\n")
	b.WriteString(m.tableHeader(cols, widths) + "\n")
	b.WriteString(m.tableRule(widths) + "\n")

	start := m.txScroll
	end := start + m.txListCapacity()
	if end > len(vis) {
		end = len(vis)
	}
	for i := start; i < end; i++ {
		row := vis[i]
		tagText, tagStyle := m.txTag(chain.ClassifyTx(row.tx, m.txWallet))

		statusCell := styledCell("✓", m.styles.Ok)
		if !row.tx.Success {
			statusCell = styledCell("✗", m.styles.Error)
		}

		cells := []tcell{
			styledCell(gasLabel(row.tx.ChainID), m.badgeStyle(row.tx.ChainID)),
			styledCell(shortHash(row.tx.Hash), m.styles.Faint),
			styledCell(tagText, tagStyle),
			txt(row.detail),
			styledCell(humanizeSince(row.tx.Timestamp), m.styles.Faint),
			statusCell,
		}
		b.WriteString(m.tableRow(cols, widths, cells, i == m.txCursor))
		b.WriteString("\n")
	}

	b.WriteString(m.txListFooter())
	return b.String()
}

// txListFooter indica si quedan más páginas por cargar o si es el fin del
// historial.
func (m Model) txListFooter() string {
	for _, id := range m.activeTxChains() {
		if !m.txExhausted[id] {
			return m.styles.Faint.Render("↓ más abajo · m carga más")
		}
	}
	return m.styles.Faint.Render("— fin del historial —")
}

// txColumns define las columnas de la tabla de transacciones. La descripción es
// la columna flex (absorbe el ancho sobrante); el momento se alinea a la derecha.
func txColumns() []column {
	return []column{
		{title: "Red", align: alignLeft, min: 5},
		{title: "Tx", align: alignLeft, min: 11},
		{title: "Tipo", align: alignLeft, min: 7},
		{title: "Detalle", align: alignLeft, min: 16, flex: true},
		{title: "Cuándo", align: alignRight, min: 8},
		{title: "Est", align: alignLeft, min: 3},
	}
}

// shortHash recorta un hash a su prefijo legible (0x + 8 dígitos + …), tolerando
// hashes más cortos de lo esperado sin romper.
func shortHash(h string) string {
	if len(h) <= 10 {
		return h
	}
	return h[:10] + "…"
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
