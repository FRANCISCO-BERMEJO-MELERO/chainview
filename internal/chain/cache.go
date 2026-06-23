package chain

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// Protección frente al rate-limit de los RPC públicos (decisión D1/D2):
//
//   - rpcTTL: ventana en la que una lectura se sirve de caché sin tocar el RPC.
//     Evita ráfagas al pulsar 'r' o cambiar de pestaña rápido.
//   - rpcCooldown: si una red devuelve 429, se marca en cooldown este tiempo;
//     mientras dure servimos el último valor cacheado en vez de insistir.
const (
	rpcTTL = 10 * time.Second
	// rpcCooldown es el mínimo que una red rate-limited queda en cooldown.
	rpcCooldown = 30 * time.Second
	// rpcCooldownJitter es el margen aleatorio que se suma al cooldown para
	// desincronizar los reintentos: si varias redes caen a la vez, no expiran
	// todas en el mismo instante provocando una nueva ráfaga (thundering herd).
	rpcCooldownJitter = 10 * time.Second
)

// cooldownDuration devuelve el cooldown con jitter: rpcCooldown + [0, jitter).
func cooldownDuration() time.Duration {
	return rpcCooldown + time.Duration(rand.Int63n(int64(rpcCooldownJitter)))
}

// errRateLimited indica que la red está rate-limited y no hay valor cacheado que
// servir.
var errRateLimited = errors.New("network rate-limited; retry in a few seconds")

// rpcEntry es un valor cacheado con su marca de tiempo.
type rpcEntry struct {
	val *big.Int
	at  time.Time
}

// BalanceAt devuelve el balance nativo (wei) con caché TTL, coalescing de
// peticiones idénticas (single-flight) y backoff ante 429. Es el acceso público;
// la lectura cruda es balanceAtRaw.
func (c *Client) BalanceAt(ctx context.Context, chainID uint64, addr common.Address) (*big.Int, error) {
	key := fmt.Sprintf("bal:%d:%s", chainID, addr.Hex())
	return c.cachedBig(ctx, key, chainID, func(ctx context.Context) (*big.Int, error) {
		return c.balanceAtRaw(ctx, chainID, addr)
	})
}

// GasPriceAt devuelve el gas price sugerido (wei) con la misma protección que
// BalanceAt. La lectura cruda es gasPriceAtRaw.
func (c *Client) GasPriceAt(ctx context.Context, chainID uint64) (*big.Int, error) {
	key := fmt.Sprintf("gas:%d", chainID)
	return c.cachedBig(ctx, key, chainID, func(ctx context.Context) (*big.Int, error) {
		return c.gasPriceAtRaw(ctx, chainID)
	})
}

// cachedBig envuelve una lectura *big.Int por RPC con TTL + single-flight +
// backoff. La clave identifica el dato (p.ej. "gas:1"); chainID se usa para el
// cooldown por red.
func (c *Client) cachedBig(ctx context.Context, key string, chainID uint64, fetch func(context.Context) (*big.Int, error)) (*big.Int, error) {
	now := time.Now()

	c.rpcMu.Lock()
	// (1) Valor fresco en caché.
	if e, ok := c.rpcCache[key]; ok && now.Sub(e.at) < rpcTTL {
		c.rpcMu.Unlock()
		c.stats.cacheHits.Add(1)
		return e.val, nil
	}
	// (2) Red en cooldown por rate-limit: servimos lo último que tengamos.
	if until, ok := c.cooldown[chainID]; ok && now.Before(until) {
		e, cached := c.rpcCache[key]
		c.rpcMu.Unlock()
		if cached {
			c.stats.staleServed.Add(1)
			return e.val, nil
		}
		return nil, errRateLimited
	}
	c.rpcMu.Unlock()

	// (3) Lectura real, coalescida: varias peticiones de la misma clave a la vez
	// comparten una sola llamada RPC (el contador sube una vez por fetch real).
	v, err, _ := c.sf.Do(key, func() (interface{}, error) {
		c.stats.rpcCalls.Add(1)
		return fetch(ctx)
	})

	if err != nil {
		if isRateLimit(err) {
			c.stats.rateLimitHits.Add(1)
			c.rpcMu.Lock()
			c.cooldown[chainID] = time.Now().Add(cooldownDuration())
			e, cached := c.rpcCache[key]
			c.rpcMu.Unlock()
			if cached {
				c.stats.staleServed.Add(1)
				return e.val, nil // servimos valor viejo durante el cooldown
			}
		}
		return nil, err
	}

	val := v.(*big.Int)
	c.rpcMu.Lock()
	c.rpcCache[key] = rpcEntry{val: val, at: time.Now()}
	delete(c.cooldown, chainID) // éxito: la red ya no está limitada
	c.rpcMu.Unlock()
	return val, nil
}

// RateLimitedChains devuelve los chain IDs actualmente en cooldown por rate-limit,
// para que la UI pueda avisarlo.
func (c *Client) RateLimitedChains() []uint64 {
	now := time.Now()
	c.rpcMu.Lock()
	defer c.rpcMu.Unlock()
	var out []uint64
	for id, until := range c.cooldown {
		if now.Before(until) {
			out = append(out, id)
		}
	}
	return out
}

// isRateLimit detecta una respuesta de rate-limit del RPC de forma best-effort
// (los providers no usan un error tipado uniforme).
func isRateLimit(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "429") ||
		strings.Contains(s, "too many requests") ||
		strings.Contains(s, "rate limit")
}
