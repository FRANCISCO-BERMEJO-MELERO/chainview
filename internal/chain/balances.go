package chain

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/sync/errgroup"
)

// maxInFlight limita cuántas peticiones RPC corren a la vez. Con publicnode
// (rate-limited) conviene no abrir cientos de conexiones de golpe: N wallets × 4
// redes crece rápido.
const maxInFlight = 8

// BalanceResult es el balance de una wallet en una red concreta (una celda de la
// tabla). Cada celda lleva su propio error: el fallo de una red no invalida las
// demás.
type BalanceResult struct {
	ChainID uint64
	Address common.Address
	Wei     *big.Int
	Err     error
}

// FetchAll consulta en paralelo el balance de cada (address × network) y
// devuelve un resultado por celda. La concurrencia está acotada por maxInFlight.
//
// Detalles importantes de concurrencia:
//   - Cada goroutine escribe en su propio índice de un slice preasignado, así que
//     no hace falta mutex (los índices son disjuntos).
//   - Las goroutines NUNCA devuelven error al errgroup: un fallo de celda se
//     guarda en BalanceResult.Err. Si devolviéramos el error, errgroup cancelaría
//     el resto de peticiones, y queremos que cada red falle de forma aislada.
//   - El context (con timeout, puesto por quien llama) acota toda la tanda: si se
//     cancela, las llamadas en vuelo se abortan y no quedan goroutines colgadas.
func (c *Client) FetchAll(ctx context.Context, addrs []common.Address, networks []Network) []BalanceResult {
	results := make([]BalanceResult, len(addrs)*len(networks))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(maxInFlight)

	i := 0
	for _, addr := range addrs {
		for _, n := range networks {
			idx := i
			results[idx] = BalanceResult{ChainID: n.ChainID, Address: addr}

			addr, chainID := addr, n.ChainID
			g.Go(func() error {
				wei, err := c.BalanceAt(ctx, chainID, addr)
				results[idx].Wei = wei
				results[idx].Err = err
				return nil // nunca cancelamos la tanda por una celda
			})
			i++
		}
	}

	_ = g.Wait() // siempre nil: los errores van por celda
	return results
}
