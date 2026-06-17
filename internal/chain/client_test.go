package chain

import (
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
)

// connect debe ser lazy y cachear: dos llamadas a la misma red devuelven el
// mismo *ethclient.Client (una sola conexión abierta).
func TestConnectCachesConnection(t *testing.T) {
	c := NewClient(DefaultNetworks())

	first, err := c.connect(ChainEthereum)
	if err != nil {
		t.Fatalf("primera conexión: %v", err)
	}
	second, err := c.connect(ChainEthereum)
	if err != nil {
		t.Fatalf("segunda conexión: %v", err)
	}
	if first != second {
		t.Fatal("connect debería reutilizar la conexión cacheada, no abrir una nueva")
	}
}

// Una red no registrada debe dar error en vez de panic.
func TestConnectUnknownChain(t *testing.T) {
	c := NewClient(DefaultNetworks())
	if _, err := c.connect(123456789); err == nil {
		t.Fatal("esperaba error para un chain id desconocido")
	}
}

// Muchas goroutines pidiendo la misma red a la vez no deben abrir conexiones
// duplicadas ni provocar data races. Ejecutar con: go test -race ./internal/chain
func TestConnectConcurrent(t *testing.T) {
	c := NewClient(DefaultNetworks())

	const goroutines = 50
	var wg sync.WaitGroup
	results := make([]*ethclient.Client, goroutines)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			conn, err := c.connect(ChainArbitrum)
			if err != nil {
				t.Errorf("goroutine %d: %v", idx, err)
				return
			}
			results[idx] = conn
		}(i)
	}
	wg.Wait()

	// Todas las goroutines deben haber recibido exactamente la misma conexión.
	for i := 1; i < goroutines; i++ {
		if results[i] != results[0] {
			t.Fatalf("se abrieron conexiones duplicadas (goroutine %d difiere)", i)
		}
	}
}
