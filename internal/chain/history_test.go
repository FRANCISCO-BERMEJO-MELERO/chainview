package chain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

const txListFixture = `{
  "status": "1",
  "message": "OK",
  "result": [
    {
      "hash": "0xaaa1",
      "from": "0x1111111111111111111111111111111111111111",
      "to":   "0x2222222222222222222222222222222222222222",
      "value": "1000000000000000000",
      "timeStamp": "1700000000",
      "isError": "0",
      "txreceipt_status": "1",
      "nonce": "5"
    },
    {
      "hash": "0xbbb2",
      "from": "0x3333333333333333333333333333333333333333",
      "to":   "0x1111111111111111111111111111111111111111",
      "value": "500000000000000000",
      "timeStamp": "1699999000",
      "isError": "1",
      "txreceipt_status": "0",
      "nonce": "6"
    }
  ]
}`

func TestParseTxList(t *testing.T) {
	txs, err := parseTxList([]byte(txListFixture))
	if err != nil {
		t.Fatalf("parseTxList: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("esperaba 2 txs, hay %d", len(txs))
	}

	first := txs[0]
	if first.Hash != "0xaaa1" {
		t.Errorf("hash = %q", first.Hash)
	}
	if first.From != common.HexToAddress("0x1111111111111111111111111111111111111111") {
		t.Errorf("from = %s", first.From.Hex())
	}
	if first.Value.String() != "1000000000000000000" {
		t.Errorf("value = %s", first.Value)
	}
	if !first.Success {
		t.Errorf("la primera tx debería ser exitosa")
	}
	if first.Nonce != 5 {
		t.Errorf("nonce = %d", first.Nonce)
	}

	// La segunda tx falló (isError=1, receipt=0).
	if txs[1].Success {
		t.Errorf("la segunda tx debería marcarse como fallida")
	}
}

func TestParseTxListEmpty(t *testing.T) {
	body := []byte(`{"status":"0","message":"No transactions found","result":[]}`)
	txs, err := parseTxList(body)
	if err != nil {
		t.Fatalf("'sin txs' no debería ser error: %v", err)
	}
	if len(txs) != 0 {
		t.Fatalf("esperaba 0 txs, hay %d", len(txs))
	}
}

func TestParseTxListAPIError(t *testing.T) {
	body := []byte(`{"status":"0","message":"NOTOK","result":"Invalid API Key"}`)
	if _, err := parseTxList(body); err == nil {
		t.Fatal("esperaba error con key inválida")
	}
}

func TestEtherscanProviderMissingKey(t *testing.T) {
	p := NewEtherscanProvider("")
	_, err := p.RecentTxs(context.Background(), 1, common.Address{}, 20)
	if err == nil {
		t.Fatal("sin API key debería devolver error legible")
	}
}

// Verifica el camino completo (URL + HTTP + parseo) contra un servidor falso,
// demostrando que TxProvider es mockeable end-to-end.
func TestEtherscanProviderLiveAgainstFake(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("chainid") != "1" {
			t.Errorf("chainid esperado 1, got %q", r.URL.Query().Get("chainid"))
		}
		if r.URL.Query().Get("apikey") != "secret" {
			t.Errorf("apikey no propagada: %q", r.URL.Query().Get("apikey"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(txListFixture))
	}))
	defer srv.Close()

	p := NewEtherscanProvider("secret")
	p.baseURL = srv.URL

	txs, err := p.RecentTxs(context.Background(), 1, common.HexToAddress("0x1111111111111111111111111111111111111111"), 20)
	if err != nil {
		t.Fatalf("RecentTxs: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("esperaba 2 txs, hay %d", len(txs))
	}
}
