package chain

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// balanceAtRaw lee el balance nativo (en wei) de una dirección directamente por
// RPC, sin caché. El acceso público con caché/single-flight/backoff es BalanceAt
// (ver cache.go). Abre la conexión de forma lazy si hace falta (ver connect).
//
// Recibe un context para poder aplicar timeout/cancelación desde quien llama:
// esto se invocará siempre desde un tea.Cmd con context.WithTimeout, de modo que
// una red lenta no cuelgue la goroutine indefinidamente. Nunca llamar desde
// Update de forma síncrona.
func (c *Client) balanceAtRaw(ctx context.Context, chainID uint64, addr common.Address) (*big.Int, error) {
	conn, err := c.connect(chainID)
	if err != nil {
		return nil, err
	}

	// nil como número de bloque = último bloque (latest).
	wei, err := conn.BalanceAt(ctx, addr, nil)
	if err != nil {
		return nil, fmt.Errorf("balance de %s en chain id %d: %w", addr.Hex(), chainID, err)
	}
	return wei, nil
}
