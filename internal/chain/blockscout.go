package chain

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// BlockscoutProvider implementa TxProvider contra las instancias públicas de
// Blockscout. Son gratuitas y SIN API key: ese es el motivo de usarlas por
// defecto (la app arranca sin que nadie tenga que registrarse). Blockscout expone
// una API compatible con la de Etherscan (module=account&action=txlist), así que
// reutilizamos el mismo parser (parseTxList) que el proveedor de Etherscan.
//
// A diferencia de Etherscan V2 (endpoint único con ?chainid=), Blockscout tiene
// un host por red; los hosts se derivan del campo BlockscoutAPI de cada Network,
// de modo que una red definida en el TOML funciona sin tocar código.
type BlockscoutProvider struct {
	hosts map[uint64]string // chain ID -> base URL del API
	http  *http.Client
}

// NewBlockscoutProvider crea el proveedor keyless tomando el host de cada red de
// su BlockscoutAPI. Las redes sin BlockscoutAPI quedan fuera (sin txs).
func NewBlockscoutProvider(networks []Network) *BlockscoutProvider {
	hosts := make(map[uint64]string, len(networks))
	for _, n := range networks {
		if n.BlockscoutAPI != "" {
			hosts[n.ChainID] = n.BlockscoutAPI
		}
	}
	return &BlockscoutProvider{
		hosts: hosts,
		http:  &http.Client{Timeout: 15 * time.Second},
	}
}

// RecentTxs consulta el historial vía Blockscout. No requiere API key.
func (p *BlockscoutProvider) RecentTxs(ctx context.Context, chainID uint64, addr common.Address, page, perPage int) ([]Tx, error) {
	base, ok := p.hosts[chainID]
	if !ok {
		return nil, fmt.Errorf("Blockscout: red no soportada (chain id %d)", chainID)
	}

	q := url.Values{}
	q.Set("module", "account")
	q.Set("action", "txlist")
	q.Set("address", addr.Hex())
	q.Set("page", strconv.Itoa(page))
	q.Set("offset", strconv.Itoa(perPage))
	q.Set("sort", "desc") // más recientes primero

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("construyendo petición a Blockscout: %w", err)
	}

	resp, err := p.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("consultando Blockscout: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("leyendo respuesta de Blockscout: %w", err)
	}
	return parseTxList(body, chainID)
}
