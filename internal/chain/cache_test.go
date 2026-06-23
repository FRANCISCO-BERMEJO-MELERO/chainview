package chain

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestIsRateLimit(t *testing.T) {
	yes := []string{"429 Too Many Requests", "rate limit exceeded", "TOO MANY REQUESTS"}
	for _, s := range yes {
		if !isRateLimit(errors.New(s)) {
			t.Errorf("isRateLimit(%q) = false, quiero true", s)
		}
	}
	if isRateLimit(errors.New("connection refused")) {
		t.Error("isRateLimit no debería detectar un error normal")
	}
	if isRateLimit(nil) {
		t.Error("isRateLimit(nil) debería ser false")
	}
}

func TestCachedBigTTL(t *testing.T) {
	c := NewClient(nil)
	var calls int32
	fetch := func(context.Context) (*big.Int, error) {
		atomic.AddInt32(&calls, 1)
		return big.NewInt(42), nil
	}

	// Dos lecturas seguidas: la segunda viene de caché (1 sola llamada real).
	for i := 0; i < 2; i++ {
		if _, err := c.cachedBig(context.Background(), "k", 1, fetch); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Errorf("llamadas = %d, quiero 1 (la 2ª debe venir de caché)", calls)
	}

	// Si la entrada caduca (la envejecemos a mano), vuelve a leer.
	c.rpcCache["k"] = rpcEntry{val: big.NewInt(42), at: time.Now().Add(-2 * rpcTTL)}
	if _, err := c.cachedBig(context.Background(), "k", 1, fetch); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Errorf("llamadas = %d, quiero 2 tras caducar el TTL", calls)
	}
}

func TestClientStats(t *testing.T) {
	c := NewClient(nil)
	fetch := func(context.Context) (*big.Int, error) { return big.NewInt(1), nil }

	// 1ª lectura: miss (cuenta como llamada RPC). 2ª: hit de caché.
	_, _ = c.cachedBig(context.Background(), "k", 1, fetch)
	_, _ = c.cachedBig(context.Background(), "k", 1, fetch)
	if s := c.Stats(); s.RPCCalls != 1 || s.CacheHits != 1 {
		t.Fatalf("tras miss+hit: RPCCalls=%d CacheHits=%d, quiero 1/1", s.RPCCalls, s.CacheHits)
	}

	// Envejecemos la entrada y forzamos un 429: cuenta el intento, el rate-limit y
	// el valor viejo servido durante el cooldown.
	c.rpcCache["k"] = rpcEntry{val: big.NewInt(1), at: time.Now().Add(-2 * rpcTTL)}
	_, _ = c.cachedBig(context.Background(), "k", 1, func(context.Context) (*big.Int, error) {
		return nil, errors.New("429 Too Many Requests")
	})
	s := c.Stats()
	if s.RPCCalls != 2 {
		t.Errorf("RPCCalls = %d, quiero 2 (incluye el intento que dio 429)", s.RPCCalls)
	}
	if s.RateLimitHits != 1 {
		t.Errorf("RateLimitHits = %d, quiero 1", s.RateLimitHits)
	}
	if s.StaleServed != 1 {
		t.Errorf("StaleServed = %d, quiero 1", s.StaleServed)
	}
}

func TestCachedBigBackoff(t *testing.T) {
	c := NewClient(nil)

	// Hay un valor viejo en caché y la red empieza a devolver 429.
	c.rpcCache["k"] = rpcEntry{val: big.NewInt(7), at: time.Now().Add(-2 * rpcTTL)}
	var calls int32
	fetch := func(context.Context) (*big.Int, error) {
		atomic.AddInt32(&calls, 1)
		return nil, errors.New("429 Too Many Requests")
	}

	// La lectura ve el 429: marca cooldown y sirve el valor viejo (sin error).
	got, err := c.cachedBig(context.Background(), "k", 1, fetch)
	if err != nil || got.Int64() != 7 {
		t.Fatalf("got=%v err=%v; quiero 7, nil (servir caché en 429)", got, err)
	}
	if rl := c.RateLimitedChains(); len(rl) != 1 || rl[0] != 1 {
		t.Errorf("RateLimitedChains = %v, quiero [1]", rl)
	}

	// Durante el cooldown no se vuelve a llamar al RPC: se sirve la caché.
	if _, err := c.cachedBig(context.Background(), "k", 1, fetch); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("llamadas = %d, quiero 1 (el cooldown evita reintentar)", calls)
	}
}

func TestCachedBigRateLimitedNoCache(t *testing.T) {
	c := NewClient(nil)
	// Red en cooldown y sin valor cacheado -> error de rate-limit.
	c.cooldown[1] = time.Now().Add(rpcCooldown)
	_, err := c.cachedBig(context.Background(), "k", 1, func(context.Context) (*big.Int, error) {
		t.Fatal("no debería llegar a leer en cooldown")
		return nil, nil
	})
	if !errors.Is(err, errRateLimited) {
		t.Errorf("err = %v, quiero errRateLimited", err)
	}
}

func TestCachedBigSingleFlight(t *testing.T) {
	c := NewClient(nil)
	var calls int32
	fetch := func(context.Context) (*big.Int, error) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(40 * time.Millisecond) // mantiene la llamada en vuelo
		return big.NewInt(1), nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.cachedBig(context.Background(), "k", 1, fetch)
		}()
	}
	wg.Wait()

	if calls != 1 {
		t.Errorf("llamadas = %d, quiero 1 (single-flight coalescen las concurrentes)", calls)
	}
}
