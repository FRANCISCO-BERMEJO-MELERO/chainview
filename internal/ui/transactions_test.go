package ui

import (
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/ethereum/go-ethereum/common"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/storage"
)

func row(chainID uint64, hash string, ago time.Duration) txRow {
	return txRow{tx: chain.Tx{ChainID: chainID, Hash: hash, Timestamp: time.Now().Add(-ago)}}
}

func TestSortDedupTxRows(t *testing.T) {
	rows := []txRow{
		row(chain.ChainEthereum, "0xa", 2*time.Hour),
		row(chain.ChainArbitrum, "0xb", 1*time.Hour),
		row(chain.ChainEthereum, "0xa", 2*time.Hour),    // duplicado (misma red+hash)
		row(chain.ChainArbitrum, "0xa", 30*time.Minute), // mismo hash, otra red: NO es duplicado
	}
	got := sortDedupTxRows(rows)
	if len(got) != 3 {
		t.Fatalf("esperaba 3 tras dedup, hay %d", len(got))
	}
	// Orden por fecha desc: el más reciente primero.
	if !got[0].tx.Timestamp.After(got[1].tx.Timestamp) || !got[1].tx.Timestamp.After(got[2].tx.Timestamp) {
		t.Error("no quedó ordenado por fecha desc")
	}
}

func TestVisibleTxsFilter(t *testing.T) {
	m := testModel(80, 24)
	m.txs = []txRow{
		row(chain.ChainEthereum, "0xa", time.Hour),
		row(chain.ChainArbitrum, "0xb", time.Hour),
		row(chain.ChainEthereum, "0xc", time.Hour),
	}
	if got := len(m.visibleTxs()); got != 3 {
		t.Errorf("sin filtro = %d, quiero 3", got)
	}
	m.txNetFilter = chain.ChainEthereum
	if got := len(m.visibleTxs()); got != 2 {
		t.Errorf("filtro ETH = %d, quiero 2", got)
	}
}

func TestNextTxFilterCycles(t *testing.T) {
	m := testModel(80, 24)
	m.networks = chain.DefaultNetworks() // 4 redes
	// 0 (todas) -> primera red -> ... -> última -> 0
	seq := []uint64{m.nextTxFilter()}
	for i := 0; i < len(m.networks); i++ {
		m.txNetFilter = seq[len(seq)-1]
		seq = append(seq, m.nextTxFilter())
	}
	if seq[0] != m.networks[0].ChainID {
		t.Errorf("primer paso = %d, quiero %d", seq[0], m.networks[0].ChainID)
	}
	if last := seq[len(seq)-1]; last != 0 {
		t.Errorf("el ciclo no vuelve a 'todas' (0), got %d", last)
	}
}

func TestClampScroll(t *testing.T) {
	viewCap := 5
	// Cursor por debajo de la ventana: la sube.
	if got := clampScroll(0, 20, viewCap, 10); got != 0 {
		t.Errorf("scroll subir = %d, quiero 0", got)
	}
	// Cursor por debajo del final visible: la baja para que entre.
	if got := clampScroll(9, 20, viewCap, 0); got != 5 {
		t.Errorf("scroll bajar = %d, quiero 5", got)
	}
	// Pocos elementos: sin desplazamiento.
	if got := clampScroll(2, 3, viewCap, 0); got != 0 {
		t.Errorf("pocos elementos = %d, quiero 0", got)
	}
}

func TestActiveTxChainsRespectsFilter(t *testing.T) {
	m := testModel(80, 24)
	m.networks = chain.DefaultNetworks()
	if got := len(m.activeTxChains()); got != len(chain.DefaultNetworks()) {
		t.Errorf("sin filtro = %d redes, quiero %d", got, len(chain.DefaultNetworks()))
	}
	m.txNetFilter = chain.ChainBase
	got := m.activeTxChains()
	if len(got) != 1 || got[0] != chain.ChainBase {
		t.Errorf("con filtro Base = %v, quiero [Base]", got)
	}
}

func TestTransactionsFrameFits(t *testing.T) {
	m := testModel(80, 24)
	m.networks = chain.DefaultNetworks()
	m.allNetworks = chain.DefaultNetworks()
	m.active = tabTransactions
	m.txState = stateLoaded
	me := common.HexToAddress("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	m.txWallet = me
	ws := &storage.Wallets{}
	_ = ws.Add(me.Hex())
	m.wallets = ws
	for i := 0; i < 30; i++ { // más filas que capacidad: fuerza scroll
		m.txs = append(m.txs, row(chain.ChainEthereum, "0x"+string(rune('a'+i%26)), time.Duration(i)*time.Hour))
	}
	out := m.renderFrame()
	if w := lipgloss.Width(out); w != 80 {
		t.Errorf("ancho = %d, quiero 80", w)
	}
	if h := lipgloss.Height(out); h != 24 {
		t.Errorf("alto = %d, quiero 24", h)
	}
}
