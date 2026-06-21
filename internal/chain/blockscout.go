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

// blockscoutHosts son las instancias públicas de Blockscout por chain ID. Son
// gratuitas y SIN API key: ese es justamente el motivo de usarlas por defecto
// (la app arranca sin que nadie tenga que registrarse en ningún sitio). A
// diferencia de Etherscan V2 (endpoint único con ?chainid=), Blockscout tiene un
// host por red. Optimism sirve su instancia bajo explorer.optimism.io.
var blockscoutHosts = map[uint64]string{
	ChainEthereum: "https://eth.blockscout.com/api",
	ChainArbitrum: "https://arbitrum.blockscout.com/api",
	ChainBase:     "https://base.blockscout.com/api",
	ChainOptimism: "https://explorer.optimism.io/api",
}

// BlockscoutProvider implementa TxProvider contra las instancias públicas de
// Blockscout. Blockscout expone una API compatible con la de Etherscan
// (module=account&action=txlist), así que reutilizamos el mismo parser
// (parseTxList) que el proveedor de Etherscan.
type BlockscoutProvider struct {
	hosts map[uint64]string // chain ID -> base URL del API
	http  *http.Client
}

// NewBlockscoutProvider crea el proveedor keyless con los hosts públicos por
// defecto.
func NewBlockscoutProvider() *BlockscoutProvider {
	return &BlockscoutProvider{
		hosts: blockscoutHosts,
		http:  &http.Client{Timeout: 15 * time.Second},
	}
}

// RecentTxs consulta el historial vía Blockscout. No requiere API key.
func (p *BlockscoutProvider) RecentTxs(ctx context.Context, chainID uint64, addr common.Address, limit int) ([]Tx, error) {
	base, ok := p.hosts[chainID]
	if !ok {
		return nil, fmt.Errorf("Blockscout: red no soportada (chain id %d)", chainID)
	}

	q := url.Values{}
	q.Set("module", "account")
	q.Set("action", "txlist")
	q.Set("address", addr.Hex())
	q.Set("page", "1")
	q.Set("offset", strconv.Itoa(limit))
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
	return parseTxList(body)
}
