package ui

import (
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/exportcsv"
)

// exportDoneMsg es el resultado de una exportación. count == 0 sin error significa
// "no había nada que exportar".
type exportDoneMsg struct {
	path  string
	count int
	err   error
}

// balanceRows construye la cabecera y las filas CSV de la vista actual de
// Balances (respeta el filtro todas/seleccionada).
func (m Model) balanceRows() ([]string, [][]string) {
	header := []string{"wallet", "ens", "chain_id", "red", "saldo", "simbolo", "error"}
	vis := m.visibleBalances()
	rows := make([][]string, 0, len(vis))
	for _, r := range vis {
		saldo, errStr := "", ""
		if r.Err != nil {
			errStr = r.Err.Error()
		} else {
			saldo = chain.FormatEther(r.Wei)
		}
		rows = append(rows, []string{
			r.Address.Hex(),
			m.ensNames[r.Address],
			strconv.FormatUint(r.ChainID, 10),
			m.networkName(r.ChainID),
			saldo,
			m.networkSymbol(r.ChainID),
			errStr,
		})
	}
	return header, rows
}

// txRows construye la cabecera y las filas CSV de la vista actual de
// Transacciones (respeta el filtro de red y todas las páginas ya cargadas).
func (m Model) txRows() ([]string, [][]string) {
	header := []string{
		"chain_id", "red", "hash", "fecha_utc", "tipo", "de", "a",
		"valor", "simbolo", "accion", "estado", "bloque", "gas_usado", "gas_price_gwei", "nonce",
	}
	vis := m.visibleTxs()
	rows := make([][]string, 0, len(vis))
	for _, row := range vis {
		tx := row.tx
		estado := "ok"
		if !tx.Success {
			estado = "fallida"
		}
		rows = append(rows, []string{
			strconv.FormatUint(tx.ChainID, 10),
			m.networkName(tx.ChainID),
			tx.Hash,
			tx.Timestamp.UTC().Format(time.RFC3339),
			chain.ClassifyTx(tx, m.txWallet).String(),
			tx.From.Hex(),
			tx.To.Hex(),
			chain.FormatEther(tx.Value),
			m.networkSymbol(tx.ChainID),
			row.detail,
			estado,
			strconv.FormatUint(tx.BlockNumber, 10),
			strconv.FormatUint(tx.GasUsed, 10),
			chain.FormatUnits(tx.GasPrice, 9),
			strconv.FormatUint(tx.Nonce, 10),
		})
	}
	return header, rows
}

// exportCmd construye un comando que escribe el CSV en background (la I/O nunca
// va en el render). Las filas se construyen ya, por valor, antes de la goroutine.
func exportCmd(name string, header []string, rows [][]string) tea.Cmd {
	return func() tea.Msg {
		if len(rows) == 0 {
			return exportDoneMsg{count: 0}
		}
		path, err := exportcsv.Write(exportcsv.Dir, name, header, rows)
		return exportDoneMsg{path: path, count: len(rows), err: err}
	}
}

// exportBalancesCmd exporta la vista actual de Balances a un CSV con fecha.
func (m Model) exportBalancesCmd() tea.Cmd {
	header, rows := m.balanceRows()
	name := "saldos-" + exportcsv.Timestamp(time.Now()) + ".csv"
	return exportCmd(name, header, rows)
}

// exportTxsCmd exporta la vista actual de Transacciones a un CSV con fecha,
// nombrado por la wallet en foco.
func (m Model) exportTxsCmd() tea.Cmd {
	header, rows := m.txRows()
	name := "txs-" + exportcsv.SanitizeName(m.displayName(m.txWallet)) + "-" + exportcsv.Timestamp(time.Now()) + ".csv"
	return exportCmd(name, header, rows)
}
