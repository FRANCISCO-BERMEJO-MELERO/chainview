package chain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestDefiLlamaPricesNativeAndToken(t *testing.T) {
	usdc := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")

	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		// El nativo de Ethereum se pide como coingecko:ethereum y el token por
		// {priceChain}:{address en minúsculas}.
		if !strings.Contains(r.URL.Path, "coingecko:ethereum") {
			t.Errorf("falta la clave del nativo en %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"coins":{
			"coingecko:ethereum":{"price":2500.5},
			"ethereum:` + strings.ToLower(usdc.Hex()) + `":{"price":1.0}
		}}`))
	}))
	defer srv.Close()

	p := NewDefiLlamaPrices(DefaultNetworks())
	p.baseURL = srv.URL + "/"

	native := PriceQuery{ChainID: ChainEthereum}
	token := PriceQuery{ChainID: ChainEthereum, Token: usdc}

	out, err := p.Prices(context.Background(), []PriceQuery{native, token})
	if err != nil {
		t.Fatalf("Prices: %v", err)
	}
	if out[native] != 2500.5 {
		t.Errorf("precio nativo = %v, quiero 2500.5", out[native])
	}
	if out[token] != 1.0 {
		t.Errorf("precio token = %v, quiero 1.0", out[token])
	}

	// Segunda llamada idéntica: se sirve de caché, sin nueva petición HTTP.
	if _, err := p.Prices(context.Background(), []PriceQuery{native, token}); err != nil {
		t.Fatalf("Prices (caché): %v", err)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Errorf("hubo %d peticiones HTTP, quiero 1 (resto de caché)", got)
	}
}

func TestDefiLlamaPricesUnknownChainIgnored(t *testing.T) {
	// Una red sin metadatos de precio no genera petición ni entra en el resultado.
	p := NewDefiLlamaPrices(DefaultNetworks())
	p.baseURL = "http://invalid.invalid/" // si se usara, fallaría
	out, err := p.Prices(context.Background(), []PriceQuery{{ChainID: 999999}})
	if err != nil {
		t.Fatalf("una red sin precio no debería dar error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("esperaba mapa vacío, got %v", out)
	}
}
