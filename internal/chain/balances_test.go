package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// fakeRPC levanta un servidor JSON-RPC de mentira que responde a cualquier
// petición con el mismo `result` (un balance hex). Evita depender de la red real
// y hace el test determinista.
func fakeRPC(result string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%q}`, req.ID, result)
	}))
}

// Ejecutar con -race para validar que no hay data races al escribir resultados.
func TestFetchAllConcurrent(t *testing.T) {
	srv := fakeRPC("0xde0b6b3a7640000") // 1e18 wei = 1 ETH
	defer srv.Close()

	networks := []Network{
		{Key: "a", Name: "A", ChainID: 1, RPCURL: srv.URL},
		{Key: "b", Name: "B", ChainID: 2, RPCURL: srv.URL},
		{Key: "c", Name: "C", ChainID: 3, RPCURL: srv.URL},
		{Key: "d", Name: "D", ChainID: 4, RPCURL: srv.URL},
	}
	c := NewClient(networks)
	defer c.Close()

	addrs := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := c.FetchAll(ctx, addrs, networks)

	if len(res) != len(addrs)*len(networks) {
		t.Fatalf("esperaba %d celdas, hay %d", len(addrs)*len(networks), len(res))
	}
	for _, r := range res {
		if r.Err != nil {
			t.Errorf("celda %s/%d: error inesperado: %v", r.Address.Hex(), r.ChainID, r.Err)
			continue
		}
		if r.Wei == nil || r.Wei.String() != "1000000000000000000" {
			t.Errorf("celda %s/%d: balance = %v, quiero 1e18", r.Address.Hex(), r.ChainID, r.Wei)
		}
	}
}

func TestFetchAllEmpty(t *testing.T) {
	c := NewClient(DefaultNetworks())
	defer c.Close()
	if res := c.FetchAll(context.Background(), nil, DefaultNetworks()); len(res) != 0 {
		t.Fatalf("sin wallets esperaba 0 celdas, hay %d", len(res))
	}
}
