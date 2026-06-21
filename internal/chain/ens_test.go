package chain

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

func TestNamehash(t *testing.T) {
	cases := map[string]string{
		"":    "0x0000000000000000000000000000000000000000000000000000000000000000",
		"eth": "0x93cdeb708b7545dc668eb9280176169d1c33cfd8ed6f04690a0bcc88a93fc4ae",
	}
	for name, want := range cases {
		if got := namehash(name).Hex(); got != want {
			t.Errorf("namehash(%q) = %s, quiero %s", name, got, want)
		}
	}
}

// ensFake responde las llamadas de ENS con valores canónicos fijos (ignora el
// node), suficiente para un único par nombre/dirección. Cuenta las llamadas para
// verificar el cacheo.
type ensFake struct {
	resolver common.Address
	addr     common.Address
	name     string
	calls    int
}

func (f *ensFake) CallContract(_ context.Context, call ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	f.calls++
	m, err := ensABI.MethodById(call.Data[:4])
	if err != nil {
		return nil, err
	}
	switch m.Name {
	case "resolver":
		return ensABI.Methods["resolver"].Outputs.Pack(f.resolver)
	case "addr":
		return ensABI.Methods["addr"].Outputs.Pack(f.addr)
	case "name":
		return ensABI.Methods["name"].Outputs.Pack(f.name)
	}
	return nil, errors.New("método inesperado")
}

func newTestResolver(caller contractCaller) *ENSResolver {
	return &ENSResolver{
		dial: func(context.Context) (contractCaller, error) { return caller, nil },
		ttl:  time.Hour,
		now:  time.Now,
		rev:  make(map[common.Address]revEntry),
		fwd:  make(map[string]fwdEntry),
	}
}

func TestResolveForward(t *testing.T) {
	target := common.HexToAddress("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	fake := &ensFake{
		resolver: common.HexToAddress("0x4976fb03C32e5B8cfe2b6cCB31c09Ba78EBaBa41"),
		addr:     target,
		name:     "vitalik.eth",
	}
	r := newTestResolver(fake)

	addr, ok := r.Resolve(context.Background(), "vitalik.eth")
	if !ok || addr != target {
		t.Fatalf("Resolve = %s, %v; quiero %s, true", addr.Hex(), ok, target.Hex())
	}

	calls := fake.calls
	if _, _ = r.Resolve(context.Background(), "VITALIK.eth"); fake.calls != calls {
		t.Errorf("la segunda resolución (normalizada) debería venir de caché, hizo %d RPC extra", fake.calls-calls)
	}
}

func TestLookupReverse(t *testing.T) {
	target := common.HexToAddress("0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045")
	fake := &ensFake{
		resolver: common.HexToAddress("0x4976fb03C32e5B8cfe2b6cCB31c09Ba78EBaBa41"),
		addr:     target, // la verificación directa exige que name->addr == target
		name:     "vitalik.eth",
	}
	r := newTestResolver(fake)

	name, ok := r.Lookup(context.Background(), target)
	if !ok || name != "vitalik.eth" {
		t.Fatalf("Lookup = %q, %v; quiero vitalik.eth, true", name, ok)
	}

	calls := fake.calls
	if _, _ = r.Lookup(context.Background(), target); fake.calls != calls {
		t.Errorf("la segunda inversa debería venir de caché, hizo %d RPC extra", fake.calls-calls)
	}
}

// zeroCaller devuelve siempre 32 bytes a cero: el registry responde "sin
// resolver", así que toda resolución es errENSNotFound.
type zeroCaller struct{ calls int }

func (z *zeroCaller) CallContract(_ context.Context, _ ethereum.CallMsg, _ *big.Int) ([]byte, error) {
	z.calls++
	return make([]byte, 32), nil
}

func TestLookupNegativeCache(t *testing.T) {
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	fake := &zeroCaller{}
	r := newTestResolver(fake)

	if _, ok := r.Lookup(context.Background(), addr); ok {
		t.Fatal("una dirección sin nombre debería dar ok=false")
	}
	calls := fake.calls
	if calls == 0 {
		t.Fatal("la primera inversa debería consultar el registry")
	}
	if _, _ = r.Lookup(context.Background(), addr); fake.calls != calls {
		t.Errorf("el negativo debería cachearse; hizo %d RPC extra", fake.calls-calls)
	}
}
