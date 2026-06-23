package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/storage"
)

// mockRPC levanta un JSON-RPC de mentira que responde a todo con el mismo result
// (un hex). Sirve para ejercitar el flujo chain→ui sin nodos reales.
func mockRPC(t *testing.T, result string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%q}`, req.ID, result)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// mockRPC429 levanta un RPC que siempre devuelve 429, para probar el camino de
// rate-limit de punta a punta.
func mockRPC429(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// mockExplorer devuelve un listado de txs de Blockscout fijo para cualquier petición.
func mockExplorer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// e2eModel construye un Model mínimo pero funcional para los tests E2E, apuntando
// a las redes/proveedores que se le pasen.
func e2eModel(client *chain.Client, txProvider chain.TxProvider, networks []chain.Network, addr common.Address) Model {
	m := testModel(80, 24)
	m.client = client
	m.txProvider = txProvider
	m.networks = networks
	m.allNetworks = networks
	ws := &storage.Wallets{}
	_ = ws.Add(addr.Hex())
	m.wallets = ws
	return m
}

// TestE2EBalancesHappyPath: Client real contra un RPC mock → fetchBalancesCmd →
// balancesMsg → Update → estado y frame correctos.
func TestE2EBalancesHappyPath(t *testing.T) {
	srv := mockRPC(t, "0xde0b6b3a7640000") // 1e18 wei = 1 ETH
	nets := []chain.Network{{Key: "a", Name: "A", ChainID: 1, Symbol: "ETH", RPCURL: srv.URL}}
	client := chain.NewClient(nets)
	defer client.Close()

	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	m := e2eModel(client, nil, nets, addr)

	msg := m.fetchBalancesCmd(context.Background(), m.loadGen)()
	r, _ := m.Update(msg)
	m = r.(Model)

	if m.balState != stateLoaded {
		t.Fatalf("balState = %v, quiero stateLoaded", m.balState)
	}
	if len(m.balResults) != 1 || m.balResults[0].Err != nil {
		t.Fatalf("balResults = %+v", m.balResults)
	}
	if got := m.balResults[0].Wei; got == nil || got.String() != "1000000000000000000" {
		t.Errorf("Wei = %v, quiero 1e18", got)
	}
	if strings.Contains(plain(m.renderFrame()), "Sin wallets") {
		t.Error("el frame no debería mostrar 'Sin wallets' tras cargar")
	}
}

// TestE2EBalances429: un RPC que devuelve 429 deja el resultado con error, dispara
// el aviso y marca la red en cooldown.
func TestE2EBalances429(t *testing.T) {
	srv := mockRPC429(t)
	nets := []chain.Network{{Key: "a", Name: "A", ChainID: 1, Symbol: "ETH", RPCURL: srv.URL}}
	client := chain.NewClient(nets)
	defer client.Close()

	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	m := e2eModel(client, nil, nets, addr)

	msg := m.fetchBalancesCmd(context.Background(), m.loadGen)()
	r, _ := m.Update(msg)
	m = r.(Model)

	bm := msg.(balancesMsg)
	if len(bm.results) != 1 || bm.results[0].Err == nil {
		t.Fatalf("esperaba un resultado con error, got %+v", bm.results)
	}
	if rl := client.RateLimitedChains(); len(rl) != 1 || rl[0] != 1 {
		t.Errorf("RateLimitedChains = %v, quiero [1]", rl)
	}
	if m.noticeLevel != noticeError {
		t.Errorf("noticeLevel = %v, quiero noticeError tras un fallo de balance", m.noticeLevel)
	}
}

// TestE2ETxsHappyPath: TxProvider real (Blockscout) contra un explorer mock →
// fetchTxPagesCmd → txPageMsg → Update → txs cargadas.
func TestE2ETxsHappyPath(t *testing.T) {
	exp := mockExplorer(t, txListE2EFixture)
	rpc := mockRPC(t, "0x0") // no se llega a usar para txs nativas, pero el client lo necesita
	nets := []chain.Network{{Key: "a", Name: "A", ChainID: 1, Symbol: "ETH", RPCURL: rpc.URL, BlockscoutAPI: exp.URL}}
	client := chain.NewClient(nets)
	defer client.Close()
	txProvider := chain.NewBlockscoutProvider(nets)

	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	m := e2eModel(client, txProvider, nets, addr)
	m.active = tabTransactions
	m.txWallet = addr
	m.txState = stateLoading
	m.txPage = map[uint64]int{}
	m.txExhausted = map[uint64]bool{}

	cmd := m.fetchTxPagesCmd(context.Background(), m.loadGen, addr, []txReq{{chainID: 1, page: 1}})
	r, _ := m.Update(cmd())
	m = r.(Model)

	if m.txState != stateLoaded {
		t.Fatalf("txState = %v, quiero stateLoaded", m.txState)
	}
	if len(m.txs) != 2 {
		t.Fatalf("esperaba 2 txs, hay %d", len(m.txs))
	}
}

const txListE2EFixture = `{
  "status": "1", "message": "OK",
  "result": [
    {"hash":"0xaaa1","from":"0x1111111111111111111111111111111111111111","to":"0x2222222222222222222222222222222222222222","value":"1000000000000000000","timeStamp":"1700000000","isError":"0","txreceipt_status":"1","nonce":"5"},
    {"hash":"0xbbb2","from":"0x3333333333333333333333333333333333333333","to":"0x1111111111111111111111111111111111111111","value":"500000000000000000","timeStamp":"1699999000","isError":"0","txreceipt_status":"1","nonce":"6"}
  ]
}`
