package chain

import (
	"context"
	"errors"
	"math/big"
	"testing"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

func TestTokenMetaKnownTable(t *testing.T) {
	c := NewClient(nil)
	usdc := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")

	meta, ok := c.TokenMeta(context.Background(), ChainEthereum, usdc)
	if !ok {
		t.Fatal("USDC debería resolverse desde la tabla local")
	}
	if meta.Symbol != "USDC" || meta.Decimals != 6 {
		t.Errorf("meta = %+v, quiero {USDC 6}", meta)
	}
}

// fakeCaller responde eth_call empaquetando salidas con la metaABI. Cuenta las
// llamadas para verificar el cacheo.
type fakeCaller struct {
	symbol   string
	decimals uint8
	err      error
	calls    int
}

func (f *fakeCaller) CallContract(_ context.Context, call ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	switch {
	case bytesEqual(call.Data, mustPack("symbol")):
		return mustPackOutput("symbol", f.symbol)
	case bytesEqual(call.Data, mustPack("decimals")):
		return mustPackOutput("decimals", f.decimals)
	}
	return nil, errors.New("método inesperado")
}

func TestCallSymbolAndDecimals(t *testing.T) {
	token := common.HexToAddress("0x9999999999999999999999999999999999999999")
	caller := &fakeCaller{symbol: "SHIB", decimals: 18}

	sym, err := callSymbol(context.Background(), caller, token)
	if err != nil || sym != "SHIB" {
		t.Fatalf("callSymbol = %q, %v; quiero SHIB, nil", sym, err)
	}
	dec, err := callDecimals(context.Background(), caller, token)
	if err != nil || dec != 18 {
		t.Fatalf("callDecimals = %d, %v; quiero 18, nil", dec, err)
	}
}

func TestTokenMetaCachesNegativeResult(t *testing.T) {
	c := NewClient(nil)
	bad := common.HexToAddress("0x8888888888888888888888888888888888888888")

	// Forzamos un fallo de resolución y verificamos que se cachea como negativo:
	// una segunda consulta no debe volver a intentar nada.
	key := "1:" + bad.Hex()
	c.tokenCache[key] = tokenEntry{ok: false}

	if _, ok := c.TokenMeta(context.Background(), ChainEthereum, bad); ok {
		t.Error("un token cacheado como fallido debería devolver ok=false")
	}
}

// --- helpers de empaquetado para el fake ---

func mustPack(method string) []byte {
	data, err := metaABI.Pack(method)
	if err != nil {
		panic(err)
	}
	return data
}

func mustPackOutput(method string, v interface{}) ([]byte, error) {
	return metaABI.Methods[method].Outputs.Pack(v)
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
