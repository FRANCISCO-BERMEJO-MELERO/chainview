package ui

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/storage"
)

// balModelWithTokens monta un Model en Balances con una wallet que tiene saldo
// nativo más dos tokens en Ethereum, y precios para tasar todo.
func balModelWithTokens() Model {
	m := testModel(100, 24)
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	usdc := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	dai := common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F")

	ws := &storage.Wallets{}
	_ = ws.Add(addr.Hex())
	m.wallets = ws

	m.balResults = []chain.BalanceResult{{
		ChainID: chain.ChainEthereum,
		Address: addr,
		Wei:     big.NewInt(1_000_000_000_000_000_000), // 1 ETH
		Tokens: []chain.TokenBalance{
			{Token: usdc, Symbol: "USDC", Decimals: 6, Balance: big.NewInt(500_000_000)},              // 500 USDC
			{Token: dai, Symbol: "DAI", Decimals: 18, Balance: big.NewInt(2_000_000_000_000_000_000)}, // 2 DAI
		},
	}}
	m.fiatCurrency = "usd"
	m.prices = map[chain.PriceQuery]float64{
		{ChainID: chain.ChainEthereum}:              2000, // ETH
		{ChainID: chain.ChainEthereum, Token: usdc}: 1,    // USDC
		{ChainID: chain.ChainEthereum, Token: dai}:  1,    // DAI
	}
	return m
}

func TestVisibleRowsFlattensNativePlusTokens(t *testing.T) {
	m := balModelWithTokens()
	rows := m.visibleRows()
	if len(rows) != 3 {
		t.Fatalf("esperaba 3 filas (nativo + 2 tokens), hay %d", len(rows))
	}
	if rows[0].token != nil {
		t.Error("la primera fila debería ser la nativa")
	}
	// Tokens ordenados por valor fiat desc: USDC (500) antes que DAI (2).
	if rows[1].token == nil || rows[1].token.Symbol != "USDC" {
		t.Errorf("fila 2 debería ser USDC (mayor valor), got %+v", rows[1].token)
	}
	if rows[2].token == nil || rows[2].token.Symbol != "DAI" {
		t.Errorf("fila 3 debería ser DAI, got %+v", rows[2].token)
	}
}

func TestVisibleFiatTotalSumsNativeAndTokens(t *testing.T) {
	m := balModelWithTokens()
	total, priced := m.visibleFiatTotal(m.visibleBalances())
	if !priced {
		t.Fatal("debería haber al menos un activo tasado")
	}
	// 1 ETH * 2000 + 500 USDC * 1 + 2 DAI * 1 = 2502.
	if total != 2502 {
		t.Errorf("total = %v, quiero 2502", total)
	}
}

func TestRenderBalancesShowsTotalAndValues(t *testing.T) {
	m := balModelWithTokens()
	out := m.renderBalances()
	if !strings.Contains(out, "Total:") {
		t.Error("la cabecera debería mostrar el total de la cartera")
	}
	if !strings.Contains(out, "$2,502.00") {
		t.Errorf("esperaba el total $2,502.00 en la salida:\n%s", out)
	}
	if !strings.Contains(out, "USDC") || !strings.Contains(out, "↳") {
		t.Error("deberían verse las filas de token con sangría")
	}
}
