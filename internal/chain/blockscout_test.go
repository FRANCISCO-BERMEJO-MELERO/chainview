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
}
