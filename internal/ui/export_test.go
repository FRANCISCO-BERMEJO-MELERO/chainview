package ui

import (
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
)

func TestBalanceRows(t *testing.T) {
	m := testModel(80, 24)
	m.networks = chain.DefaultNetworks()
	addr := common.HexToAddress("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	m.ensNames[addr] = "vitalik.eth"
	m.balResults = []chain.BalanceResult{
		{ChainID: chain.ChainEthereum, Address: addr, Wei: big.NewInt(1500000000000000000)},
		{ChainID: chain.ChainBase, Address: addr, Err: errTest{}},
	}

	header, rows := m.balanceRows()
	if len(header) != 7 {
		t.Fatalf("cabecera con %d columnas, quiero 7", len(header))
	}
	if len(rows) != 2 {
		t.Fatalf("esperaba 2 filas, hay %d", len(rows))
	}
	// Fila OK: ens y saldo presentes, error vacío.
	if rows[0][1] != "vitalik.eth" || rows[0][4] != "1.5" || rows[0][6] != "" {
		t.Errorf("fila OK inesperada: %v", rows[0])
	}
	// Fila con error: saldo vacío, error con texto.
	if rows[1][4] != "" || rows[1][6] == "" {
		t.Errorf("fila con error inesperada: %v", rows[1])
	}
}

func TestTxRows(t *testing.T) {
	m := testModel(80, 24)
	m.networks = chain.DefaultNetworks()
	me := common.HexToAddress("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	other := common.HexToAddress("0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B")
	m.txWallet = me
	m.txs = []txRow{
		{tx: chain.Tx{ChainID: chain.ChainEthereum, Hash: "0xabc", From: me, To: other, Value: big.NewInt(1e18), Success: true, Timestamp: time.Unix(1700000000, 0), GasPrice: big.NewInt(0)}, detail: "→ 0xAb58…  1 ETH"},
	}

	header, rows := m.txRows()
	if len(header) != 15 {
		t.Fatalf("cabecera con %d columnas, quiero 15", len(header))
	}
	if len(rows) != 1 {
		t.Fatalf("esperaba 1 fila, hay %d", len(rows))
	}
	r := rows[0]
	if r[2] != "0xabc" {
		t.Errorf("hash = %q", r[2])
	}
	if r[4] != "OUT" { // tipo: saliente nativa
		t.Errorf("tipo = %q, quiero OUT", r[4])
	}
	if r[10] != "ok" {
		t.Errorf("estado = %q, quiero ok", r[10])
	}
}

// errTest es un error de prueba con mensaje no vacío.
type errTest struct{}

func (errTest) Error() string { return "timeout" }
