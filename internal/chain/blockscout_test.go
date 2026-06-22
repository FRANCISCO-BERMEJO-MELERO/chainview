package chain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// Verifica el camino completo del proveedor keyless (URL sin apikey + HTTP +
// parseo compartido) contra un servidor falso que imita a Blockscout.
func TestBlockscoutProviderLiveAgainstFake(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("action") != "txlist" {
			t.Errorf("action esperado txlist, got %q", q.Get("action"))
		}
		if q.Has("apikey") {
			t.Errorf("Blockscout no debería enviar apikey, got %q", q.Get("apikey"))
		}
		if q.Get("offset") != "20" {
			t.Errorf("offset esperado 20, got %q", q.Get("offset"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(txListFixture))
	}))
	defer srv.Close()

	p := NewBlockscoutProvider(DefaultNetworks())
	p.hosts = map[uint64]string{ChainEthereum: srv.URL}

	txs, err := p.RecentTxs(context.Background(), ChainEthereum, common.HexToAddress("0x1111111111111111111111111111111111111111"), 1, 20)
	if err != nil {
		t.Fatalf("RecentTxs: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("esperaba 2 txs, hay %d", len(txs))
	}
}

func TestBlockscoutProviderUnsupportedChain(t *testing.T) {
	p := NewBlockscoutProvider(DefaultNetworks())
	if _, err := p.RecentTxs(context.Background(), 99999, common.Address{}, 1, 20); err == nil {
		t.Fatal("una red no soportada debería devolver error")
	}
	if _, err := p.TokenBalances(context.Background(), 99999, common.Address{}); err == nil {
		t.Fatal("TokenBalances en red no soportada debería devolver error")
	}
}

// tokenListFixture mezcla un ERC-20 válido, un ERC-20 con saldo cero, un NFT
// (ERC-721) y un token sin decimales legibles: solo el primero debe pasar.
const tokenListFixture = `{"status":"1","message":"OK","result":[
	{"contractAddress":"0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48","symbol":"USDC","decimals":"6","balance":"1500000","type":"ERC-20"},
	{"contractAddress":"0x0000000000000000000000000000000000000001","symbol":"ZERO","decimals":"18","balance":"0","type":"ERC-20"},
	{"contractAddress":"0x0000000000000000000000000000000000000002","symbol":"NFT","decimals":"0","balance":"1","type":"ERC-721"},
	{"contractAddress":"0x0000000000000000000000000000000000000003","symbol":"BAD","decimals":"abc","balance":"100","type":"ERC-20"}
]}`

func TestBlockscoutTokenBalancesFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("action") != "tokenlist" {
			t.Errorf("action esperado tokenlist, got %q", r.URL.Query().Get("action"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(tokenListFixture))
	}))
	defer srv.Close()

	p := NewBlockscoutProvider(DefaultNetworks())
	p.hosts = map[uint64]string{ChainEthereum: srv.URL}

	toks, err := p.TokenBalances(context.Background(), ChainEthereum, common.HexToAddress("0x1111111111111111111111111111111111111111"))
	if err != nil {
		t.Fatalf("TokenBalances: %v", err)
	}
	if len(toks) != 1 {
		t.Fatalf("esperaba 1 token tras filtrar, hay %d: %+v", len(toks), toks)
	}
	if toks[0].Symbol != "USDC" || toks[0].Decimals != 6 || toks[0].Balance.String() != "1500000" {
		t.Errorf("token mal parseado: %+v", toks[0])
	}
}
