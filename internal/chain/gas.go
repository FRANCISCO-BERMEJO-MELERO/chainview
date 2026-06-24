package chain

import (
	"context"
	"fmt"
	"math/big"

	"golang.org/x/sync/errgroup"
)

// GasResult es el gas price sugerido de una red (en wei). Como en BalanceResult,
// cada red lleva su propio error para que el fallo de una no invalide las demás.
type GasResult struct {
	ChainID uint64
	Wei     *big.Int
	Err     error
}

// gasPriceAtRaw lee el gas price sugerido de una red (eth_gasPrice), en wei, sin
// caché. El acceso público con caché/single-flight/backoff es GasPriceAt (ver
// cache.go). Abre la conexión de forma lazy si hace falta.
func (c *Client) gasPriceAtRaw(ctx context.Context, chainID uint64) (*big.Int, error) {
	conn, err := c.connect(chainID)
	if err != nil {
		return nil, err
	}
	wei, err := conn.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("gas price on chain id %d: %w", chainID, err)
	}
	return wei, nil
}

// GasPrices consulta en paralelo el gas price de cada red y devuelve un resultado
// por red. Mismo patrón de concurrencia que FetchAll: cada goroutine escribe en su
// propio índice, los errores van por celda (nunca cancelan la tanda) y el context
// acota toda la operación.
func (c *Client) GasPrices(ctx context.Context, networks []Network) []GasResult {
	results := make([]GasResult, len(networks))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxInFlight)

	for i, n := range networks {
		idx, chainID := i, n.ChainID
		results[idx] = GasResult{ChainID: chainID}
		g.Go(func() error {
			wei, err := c.GasPriceAt(ctx, chainID)
			results[idx].Wei = wei
			results[idx].Err = err
			return nil // nunca cancelamos la tanda por una red
		})
	}

	_ = g.Wait() // siempre nil: los errores van por celda
	return results
}
