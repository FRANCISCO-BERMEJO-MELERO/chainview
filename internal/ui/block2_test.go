package ui

import (
	"math/big"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/ethereum/go-ethereum/common"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/storage"
)

// keyMsg construye un KeyPressMsg para los tests del bloque 2.
func keyMsg(s string) tea.KeyPressMsg {
	switch s {
	case "ctrl+d":
		return tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'd'}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	default:
		return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
	}
}

// --- 2.8 confirmación de borrado ---

func TestConfirmDeleteRequiresTwoPresses(t *testing.T) {
	if keyMsg("ctrl+d").String() != "ctrl+d" {
		t.Fatalf("el helper de teclas no produce ctrl+d, got %q", keyMsg("ctrl+d").String())
	}
	ws := &storage.Wallets{}
	_ = ws.Add("0x1111111111111111111111111111111111111111")
	m := testModel(80, 24)
	m.wallets = ws
	m.active = tabAccounts

	// Primer ctrl+d: arma la confirmación, no borra.
	r1, _ := m.updateAccounts(keyMsg("ctrl+d"))
	m1 := r1.(Model)
	if !m1.confirmDel || m1.wallets.Len() != 1 {
		t.Fatalf("primer ctrl+d: confirmDel=%v len=%d, esperaba true/1", m1.confirmDel, m1.wallets.Len())
	}
	// esc cancela.
	r2, _ := m1.updateAccounts(keyMsg("esc"))
	m2 := r2.(Model)
	if m2.confirmDel || m2.wallets.Len() != 1 {
		t.Fatalf("esc debería cancelar la confirmación sin borrar")
	}
	// Dos ctrl+d seguidos: borra.
	r3, _ := m2.updateAccounts(keyMsg("ctrl+d"))
	r4, _ := r3.(Model).updateAccounts(keyMsg("ctrl+d"))
	if got := r4.(Model).wallets.Len(); got != 0 {
		t.Errorf("dos ctrl+d deberían borrar la wallet, quedan %d", got)
	}
}

// --- 2.3 ordenación ---

func TestSortBalancesByValueDesc(t *testing.T) {
	m := balModelWithTokens() // de balances_test.go: 1 ETH = $2000 en Ethereum
	// Añadimos una segunda celda de menor valor en otra red.
	low := common.HexToAddress("0x2222222222222222222222222222222222222222")
	m.balResults = append(m.balResults, chain.BalanceResult{
		ChainID: chain.ChainBase, Address: low, Wei: big.NewInt(1_000_000_000_000_000), // 0.001 ETH
	})
	m.prices[chain.PriceQuery{ChainID: chain.ChainBase}] = 2000
	m.balSortKey = 1 // valor
	m.balSortAsc = false

	vis := append([]chain.BalanceResult(nil), m.balResults...)
	m.sortBalances(vis)
	if vis[0].ChainID != chain.ChainEthereum {
		t.Errorf("orden por valor desc: esperaba Ethereum (mayor) primero, got chain %d", vis[0].ChainID)
	}
}

func TestSortName(t *testing.T) {
	if balSortName(1) != "valor" || balSortName(0) != "" {
		t.Error("balSortName mal mapeado")
	}
	if txSortName(2) != "valor" || txSortName(0) != "fecha" {
		t.Error("txSortName mal mapeado")
	}
}

// --- 2.4 / 2.5 copiar y abrir ---

func TestBalCopyTarget(t *testing.T) {
	wallet := common.HexToAddress("0x1111111111111111111111111111111111111111")
	token := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	if v, _ := balCopyTarget(balRow{address: wallet}); v != wallet.Hex() {
		t.Errorf("fila nativa debería copiar la address de la wallet, got %q", v)
	}
	tb := chain.TokenBalance{Token: token}
	if v, _ := balCopyTarget(balRow{address: wallet, token: &tb}); v != token.Hex() {
		t.Errorf("fila de token debería copiar la address del token, got %q", v)
	}
}

func TestExplorerURLs(t *testing.T) {
	m := testModel(80, 24) // networks = DefaultNetworks(), con Explorer poblado
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	got, ok := m.explorerAddressURL(chain.ChainEthereum, addr)
	if !ok || !strings.HasPrefix(got, "https://etherscan.io/address/0x") {
		t.Errorf("URL de address mal construida: %q (ok=%v)", got, ok)
	}
	txURL, ok := m.explorerTxURL(chain.ChainEthereum, "0xabc")
	if !ok || txURL != "https://etherscan.io/tx/0xabc" {
		t.Errorf("URL de tx mal construida: %q", txURL)
	}
	// Red sin explorador → false.
	m.networks = []chain.Network{{ChainID: 999}}
	m.allNetworks = m.networks
	if _, ok := m.explorerAddressURL(999, addr); ok {
		t.Error("una red sin Explorer no debería dar URL")
	}
}
