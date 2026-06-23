package ui

import (
	"context"
	"math/big"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/ethereum/go-ethereum/common"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
)

// balancesMsg transporta el resultado del fetch concurrente de balances. gen es la
// generación de carga (3.6): si no coincide con la actual, el resultado se descarta.
type balancesMsg struct {
	gen     int
	results []chain.BalanceResult
}

// pricesMsg transporta los precios fiat tasados para los activos en pantalla.
type pricesMsg struct {
	gen    int
	prices map[chain.PriceQuery]float64
}

// fetchPricesCmd tasa en fiat los activos presentes en los balances cargados. Va
// en background y captura el proveedor por valor; un fallo de tasación no rompe la
// tabla (las celdas sin precio muestran "—").
func (m Model) fetchPricesCmd(ctx context.Context, gen int) tea.Cmd {
	provider := m.priceProvider
	qs := m.priceQueries()
	if provider == nil || len(qs) == 0 {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		prices, _ := provider.Prices(ctx, qs)
		return pricesMsg{gen: gen, prices: prices}
	}
}

// priceQueries reúne, sin duplicados, los activos a tasar a partir de los balances
// actuales: el nativo de cada red presente (y, con 1.2, los tokens de cada celda).
func (m Model) priceQueries() []chain.PriceQuery {
	seen := map[chain.PriceQuery]struct{}{}
	var qs []chain.PriceQuery
	add := func(q chain.PriceQuery) {
		if _, ok := seen[q]; ok {
			return
		}
		seen[q] = struct{}{}
		qs = append(qs, q)
	}
	for _, r := range m.balResults {
		add(chain.PriceQuery{ChainID: r.ChainID}) // nativo
		for _, t := range r.Tokens {
			add(chain.PriceQuery{ChainID: r.ChainID, Token: t.Token})
		}
	}
	return qs
}

// refreshTickMsg lo emite el tick periódico de refresco.
type refreshTickMsg struct{}

// refreshTickCmd programa el siguiente tick de refresco (decisión D2: polling).
func refreshTickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return refreshTickMsg{} })
}

// fetchBalancesCmd lanza el fetch concurrente (wallets × redes) en background.
// Captura las dependencias por valor para no cerrar sobre el Model completo.
func (m Model) fetchBalancesCmd(ctx context.Context, gen int) tea.Cmd {
	client := m.client
	addrs := m.wallets.List()
	networks := m.networks
	tokens := m.tokenProvider
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		return balancesMsg{gen: gen, results: client.FetchAll(ctx, addrs, networks, tokens)}
	}
}

// updateBalances maneja las teclas de la pestaña Balances: navegar la tabla y
// recargar manualmente.
func (m Model) updateBalances(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		if m.balCursor > 0 {
			m.balCursor--
		}
	case "down":
		if m.balCursor < len(m.visibleRows())-1 {
			m.balCursor++
		}
	case "f":
		// Alterna entre ver todas las wallets y solo la seleccionada en Cuentas.
		m.balFocus = !m.balFocus
		m.balCursor = 0
	case "s":
		m.balSortKey = (m.balSortKey + 1) % 4 // catálogo → valor → red → wallet → …
		m.balCursor = 0
	case "S", "shift+s":
		m.balSortAsc = !m.balSortAsc
		m.balCursor = 0
	case "e":
		return m, m.exportBalancesCmd()
	case "y":
		if row, ok := m.selectedBalRow(); ok {
			val, label := balCopyTarget(row)
			return m, copyToClipboardCmd(val, label)
		}
	case "o":
		if row, ok := m.selectedBalRow(); ok {
			addr := row.address
			if row.token != nil {
				addr = row.token.Token
			}
			if url, ok := m.explorerAddressURL(row.chainID, addr); ok {
				return m, openURLCmd(url)
			}
			m.setNotice(noticeInfo, "Esta red no tiene explorador configurado")
			return m, noticeClearCmd(m.noticeUntil)
		}
	case "r":
		if m.balState != stateLoading && m.wallets.Len() > 0 {
			m.balState = stateLoading
			ctx, gen := m.nextLoad()
			return m, tea.Batch(m.spinner.Tick, m.fetchBalancesCmd(ctx, gen))
		}
	}
	return m, nil
}

// selectedBalRow devuelve la fila (nativo o token) bajo el cursor de Balances.
func (m Model) selectedBalRow() (balRow, bool) {
	rows := m.visibleRows()
	if m.balCursor < 0 || m.balCursor >= len(rows) {
		return balRow{}, false
	}
	return rows[m.balCursor], true
}

// balCopyTarget devuelve el dato a copiar de una fila y su etiqueta: la address de
// la wallet (fila nativa) o la del contrato del token (fila de token).
func balCopyTarget(row balRow) (value, label string) {
	if row.token != nil {
		return row.token.Token.Hex(), "address del token"
	}
	return row.address.Hex(), "address"
}

// visibleBalances son los balances que se muestran: todos, o solo los de la
// wallet seleccionada cuando está activo el foco individual.
func (m Model) visibleBalances() []chain.BalanceResult {
	if !m.balFocus {
		return m.balResults
	}
	wallet, ok := m.selectedWallet()
	if !ok {
		return m.balResults
	}
	out := make([]chain.BalanceResult, 0, len(m.networks))
	for _, r := range m.balResults {
		if r.Address == wallet {
			out = append(out, r)
		}
	}
	return out
}

// countBalanceErrors cuenta las celdas con error en una tanda de balances.
func countBalanceErrors(results []chain.BalanceResult) int {
	n := 0
	for _, r := range results {
		if r.Err != nil {
			n++
		}
	}
	return n
}

func (m *Model) clampBalCursor() {
	if m.balCursor >= len(m.visibleRows()) {
		m.balCursor = len(m.visibleRows()) - 1
	}
	if m.balCursor < 0 {
		m.balCursor = 0
	}
}

// balRow es una fila aplanada de la tabla de Balances: o el saldo nativo de una
// wallet en una red, o uno de sus tokens ERC-20 (token != nil). El cursor indexa
// esta lista plana.
type balRow struct {
	address common.Address
	chainID uint64
	wei     *big.Int            // saldo nativo (solo si token == nil)
	err     error               // error del nativo (solo si token == nil)
	token   *chain.TokenBalance // si != nil, la fila es de un ERC-20
}

// visibleRows aplana los balances visibles a filas: cada wallet×red aporta su
// fila nativa seguida de sus tokens, ordenados por valor fiat desc (los tasables
// arriba) y, a igualdad, por símbolo (orden que ya trae el provider).
func (m Model) visibleRows() []balRow {
	// Copiamos antes de ordenar: visibleBalances puede devolver m.balResults tal
	// cual y no queremos mutar el orden original.
	vis := append([]chain.BalanceResult(nil), m.visibleBalances()...)
	m.sortBalances(vis)
	rows := make([]balRow, 0, len(vis))
	for _, r := range vis {
		r := r
		rows = append(rows, balRow{address: r.Address, chainID: r.ChainID, wei: r.Wei, err: r.Err})

		toks := append([]chain.TokenBalance(nil), r.Tokens...)
		sort.SliceStable(toks, func(i, j int) bool {
			vi, _ := m.fiatValue(chain.PriceQuery{ChainID: r.ChainID, Token: toks[i].Token}, toks[i].Balance, toks[i].Decimals)
			vj, _ := m.fiatValue(chain.PriceQuery{ChainID: r.ChainID, Token: toks[j].Token}, toks[j].Balance, toks[j].Decimals)
			return vi > vj
		})
		for k := range toks {
			t := toks[k]
			rows = append(rows, balRow{address: r.Address, chainID: r.ChainID, token: &t})
		}
	}
	return rows
}

// sortBalances ordena en sitio las celdas visibles según balSortKey/balSortAsc
// (2.3). Clave 0 = orden de catálogo (no toca). Ordena a nivel de wallet×red; los
// tokens siguen colgando de su nativo al aplanar después.
func (m Model) sortBalances(vis []chain.BalanceResult) {
	if m.balSortKey == 0 {
		return
	}
	less := func(i, j int) bool {
		switch m.balSortKey {
		case 1: // valor fiat del nativo
			vi, _ := m.fiatValue(chain.PriceQuery{ChainID: vis[i].ChainID}, vis[i].Wei, 18)
			vj, _ := m.fiatValue(chain.PriceQuery{ChainID: vis[j].ChainID}, vis[j].Wei, 18)
			return vi < vj
		case 2: // red
			return m.networkName(vis[i].ChainID) < m.networkName(vis[j].ChainID)
		case 3: // wallet
			return m.displayName(vis[i].Address) < m.displayName(vis[j].Address)
		}
		return false
	}
	sort.SliceStable(vis, func(i, j int) bool {
		if m.balSortAsc {
			return less(i, j)
		}
		return less(j, i)
	})
}

// sortKeyName traduce una clave de orden a su etiqueta para el indicador.
func balSortName(key int) string {
	switch key {
	case 1:
		return "valor"
	case 2:
		return "red"
	case 3:
		return "wallet"
	}
	return ""
}

// balanceColumns define las columnas de la tabla de balances. El espaciador flex
// empuja el balance (número + símbolo) contra el borde derecho, de modo que los
// importes queden alineados a la derecha y sean fáciles de comparar.
func balanceColumns() []column {
	return []column{
		{title: "Wallet", align: alignLeft, min: 16},
		{title: "Red", align: alignLeft, min: 12},
		{title: "", align: alignLeft, min: 1, flex: true}, // espaciador
		{title: "Balance", align: alignRight, min: 12},
		{title: "", align: alignLeft, min: 5},        // símbolo
		{title: "Valor", align: alignRight, min: 12}, // valor fiat (1.1)
	}
}

// weiToFloat convierte un entero en unidades mínimas a su valor decimal (float),
// para multiplicarlo por un precio fiat. Es solo para presentación: una pérdida de
// precisión de coma flotante en el último céntimo es irrelevante aquí.
func weiToFloat(v *big.Int, decimals uint8) float64 {
	if v == nil || v.Sign() == 0 {
		return 0
	}
	f := new(big.Float).SetInt(v)
	div := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	out, _ := new(big.Float).Quo(f, div).Float64()
	return out
}

// fiatValue devuelve el valor fiat de una cantidad y si se pudo tasar (hay precio).
func (m Model) fiatValue(q chain.PriceQuery, amount *big.Int, decimals uint8) (float64, bool) {
	price, ok := m.prices[q]
	if !ok {
		return 0, false
	}
	return weiToFloat(amount, decimals) * price, true
}

// fiatCell construye la celda de valor: el importe en fiat si hay precio, o un
// guion atenuado si el activo aún no se ha podido tasar.
func (m Model) fiatCell(q chain.PriceQuery, amount *big.Int, decimals uint8) tcell {
	if v, ok := m.fiatValue(q, amount, decimals); ok {
		return styledCell(chain.FormatFiat(v, m.fiatCurrency), m.styles.Balance)
	}
	return styledCell("—", m.styles.Faint)
}

// visibleFiatTotal suma el valor fiat de las celdas visibles que se han podido
// tasar (nativo + tokens). El segundo valor es false si no se tasó ninguna, para
// no mostrar un "Total: $0.00" engañoso antes de que lleguen los precios.
func (m Model) visibleFiatTotal(vis []chain.BalanceResult) (float64, bool) {
	var total float64
	priced := false
	for _, r := range vis {
		if r.Err == nil {
			if v, ok := m.fiatValue(chain.PriceQuery{ChainID: r.ChainID}, r.Wei, 18); ok {
				total += v
				priced = true
			}
		}
		for _, t := range r.Tokens {
			if v, ok := m.fiatValue(chain.PriceQuery{ChainID: r.ChainID, Token: t.Token}, t.Balance, t.Decimals); ok {
				total += v
				priced = true
			}
		}
	}
	return total, priced
}

func (m Model) renderBalances() string {
	if m.wallets.Len() == 0 {
		return m.renderState("◯", "Sin wallets", "Añádelas en la pestaña Cuentas para ver balances")
	}
	// Primera carga sin datos previos: estado de carga centrado.
	if m.balState == stateLoading && len(m.balResults) == 0 {
		return m.renderState(m.spinner.View(), "Cargando balances…", "")
	}

	cols := balanceColumns()
	widths := layoutColumns(cols, m.contentW)
	vis := m.visibleBalances()

	// Contexto: modo de vista (todas / wallet seleccionada).
	scope := "todas las wallets"
	if m.balFocus {
		if w, ok := m.selectedWallet(); ok {
			scope = m.displayName(w)
		}
	}
	ctx := m.styles.Balance.Render("Saldos") + m.styles.Faint.Render(" · "+scope)
	if name := balSortName(m.balSortKey); name != "" {
		arrow := "↓"
		if m.balSortAsc {
			arrow = "↑"
		}
		ctx += m.styles.Faint.Render(" · orden: " + name + " " + arrow)
	}
	if m.balState == stateLoading {
		ctx += m.styles.Faint.Render("  " + m.spinner.View())
	}

	// Total de la cartera (1.1): suma del valor fiat de las celdas visibles que se
	// han podido tasar. Se alinea a la derecha de la línea de contexto.
	total, anyPriced := m.visibleFiatTotal(vis)
	if anyPriced {
		totalStr := m.styles.Faint.Render("Total: ") +
			m.styles.Balance.Render(chain.FormatFiat(total, m.fiatCurrency))
		ctx = bar(ctx, totalStr, m.contentW)
	}

	var b strings.Builder
	b.WriteString(ctx + "\n\n")
	b.WriteString(m.tableHeader(cols, widths) + "\n")
	b.WriteString(m.tableRule(widths) + "\n")

	for i, row := range m.visibleRows() {
		cells := m.balanceCells(row)
		b.WriteString(m.tableRow(cols, widths, cells, i == m.balCursor))
		b.WriteString("\n")
	}
	return b.String()
}

// balanceCells construye las celdas de una fila de Balances, ya sea el saldo
// nativo o un token ERC-20 (que cuelga del nativo con sangría).
func (m Model) balanceCells(row balRow) []tcell {
	if row.token != nil {
		t := row.token
		amount := chain.FormatUnits(t.Balance, int(t.Decimals))
		valueCell := m.fiatCell(chain.PriceQuery{ChainID: row.chainID, Token: t.Token}, t.Balance, t.Decimals)
		return []tcell{
			txt(""), // bajo la wallet
			styledCell("  ↳ token", m.styles.Faint),
			txt(""), // espaciador
			styledCell(amount, m.styles.Balance),
			styledCell(t.Symbol, m.styles.Symbol),
			valueCell,
		}
	}

	// Importe a la derecha + símbolo como dato secundario; en error, un guion
	// neutro y "error" en rojo en la columna del símbolo.
	amountCell := styledCell(chain.FormatEther(row.wei), m.styles.Balance)
	symbolCell := styledCell(m.networkSymbol(row.chainID), m.styles.Symbol)
	valueCell := m.fiatCell(chain.PriceQuery{ChainID: row.chainID}, row.wei, 18)
	if row.err != nil {
		amountCell = styledCell("—", m.styles.Faint)
		symbolCell = styledCell("error", m.styles.Error)
		valueCell = styledCell("", m.styles.Faint)
	}
	return []tcell{
		txt(m.displayName(row.address)),
		styledCell(m.networkName(row.chainID), m.styles.Faint),
		txt(""), // espaciador
		amountCell,
		symbolCell,
		valueCell,
	}
}

// networkName resuelve el nombre legible de una red por su chain ID.
func (m Model) networkName(chainID uint64) string {
	for _, n := range m.networks {
		if n.ChainID == chainID {
			return n.Name
		}
	}
	return "?"
}

// networkSymbol resuelve el símbolo de la moneda nativa de una red.
func (m Model) networkSymbol(chainID uint64) string {
	for _, n := range m.networks {
		if n.ChainID == chainID {
			return n.Symbol
		}
	}
	return ""
}

// shortAddr acorta una dirección a la forma 0x1234…abcd.
func shortAddr(a common.Address) string {
	h := a.Hex()
	return h[:6] + "…" + h[len(h)-4:]
}

// fit ajusta un string a un ancho fijo: trunca con … si sobra, o rellena con
// espacios si falta. Sirve para alinear columnas de la tabla.
func fit(s string, w int) string {
	r := []rune(s)
	if len(r) > w {
		if w <= 1 {
			return string(r[:w])
		}
		return string(r[:w-1]) + "…"
	}
	return s + strings.Repeat(" ", w-len(r))
}
