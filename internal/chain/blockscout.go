package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// maxTokensPerCell acota cuántos ERC-20 mostramos por wallet×red: las wallets
// reales acumulan mucho token de spam, y sin tope la tabla se vuelve ilegible.
const maxTokensPerCell = 25

// tokenListTTL es la ventana de caché del descubrimiento de tokens (HTTP, no RPC).
const tokenListTTL = 30 * time.Second

// BlockscoutProvider implementa TxProvider (y el descubrimiento de tokens, 1.2)
// contra las instancias públicas de Blockscout. Son gratuitas y SIN API key: ese
// es el motivo de usarlas por defecto (la app arranca sin que nadie tenga que
// registrarse). Blockscout expone una API compatible con la de Etherscan
// (module=account&action=txlist), así que reutilizamos el mismo parser
// (parseTxList) que el proveedor de Etherscan.
//
// A diferencia de Etherscan V2 (endpoint único con ?chainid=), Blockscout tiene
// un host por red; los hosts se derivan del campo BlockscoutAPI de cada Network,
// de modo que una red definida en el TOML funciona sin tocar código.
type BlockscoutProvider struct {
	hosts map[uint64]string // chain ID -> base URL del API
	http  *http.Client

	tokMu    sync.Mutex
	tokCache map[string]tokListEntry // "chainID:addr" -> tokens cacheados con TTL
}

// tokListEntry es un descubrimiento de tokens cacheado con su marca de tiempo.
type tokListEntry struct {
	tokens []TokenBalance
	at     time.Time
}

// NewBlockscoutProvider crea el proveedor keyless tomando el host de cada red de
// su BlockscoutAPI. Las redes sin BlockscoutAPI quedan fuera (sin txs ni tokens).
func NewBlockscoutProvider(networks []Network) *BlockscoutProvider {
	hosts := make(map[uint64]string, len(networks))
	for _, n := range networks {
		if n.BlockscoutAPI != "" {
			hosts[n.ChainID] = n.BlockscoutAPI
		}
	}
	return &BlockscoutProvider{
		hosts:    hosts,
		http:     &http.Client{Timeout: 15 * time.Second},
		tokCache: map[string]tokListEntry{},
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

// TokenBalances descubre los ERC-20 que una wallet tiene en una red, vía
// Blockscout (action=tokenlist), sin API key. Filtra a ERC-20 con saldo positivo,
// ordena por símbolo y acota a maxTokensPerCell. Cachea el resultado con TTL para
// no consultar en cada refresco.
func (p *BlockscoutProvider) TokenBalances(ctx context.Context, chainID uint64, addr common.Address) ([]TokenBalance, error) {
	base, ok := p.hosts[chainID]
	if !ok {
		return nil, fmt.Errorf("Blockscout: red no soportada (chain id %d)", chainID)
	}

	key := fmt.Sprintf("%d:%s", chainID, addr.Hex())
	p.tokMu.Lock()
	if e, ok := p.tokCache[key]; ok && time.Since(e.at) < tokenListTTL {
		p.tokMu.Unlock()
		return e.tokens, nil
	}
	p.tokMu.Unlock()

	q := url.Values{}
	q.Set("module", "account")
	q.Set("action", "tokenlist")
	q.Set("address", addr.Hex())

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
	tokens, err := parseTokenList(body)
	if err != nil {
		return nil, err
	}

	p.tokMu.Lock()
	p.tokCache[key] = tokListEntry{tokens: tokens, at: time.Now()}
	p.tokMu.Unlock()
	return tokens, nil
}

// blockscoutToken es una entrada de la respuesta de tokenlist (todo en strings,
// formato estilo Etherscan).
type blockscoutToken struct {
	ContractAddress string `json:"contractAddress"`
	Symbol          string `json:"symbol"`
	Decimals        string `json:"decimals"`
	Balance         string `json:"balance"`
	Type            string `json:"type"`
}

// parseTokenList traduce el cuerpo JSON de tokenlist a []TokenBalance: solo
// ERC-20 con saldo positivo y decimales válidos, ordenados por símbolo y acotados
// a maxTokensPerCell. Comparte el sobre status/message/result con parseTxList.
func parseTokenList(body []byte) ([]TokenBalance, error) {
	var resp etherscanResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("respuesta de Blockscout ilegible: %w", err)
	}
	if resp.Status != "1" {
		// status "0" con result vacío = la wallet no tiene tokens (no es error).
		return []TokenBalance{}, nil
	}

	var raw []blockscoutToken
	if err := json.Unmarshal(resp.Result, &raw); err != nil {
		return nil, fmt.Errorf("lista de tokens ilegible: %w", err)
	}

	out := make([]TokenBalance, 0, len(raw))
	for _, r := range raw {
		if !strings.EqualFold(r.Type, "ERC-20") {
			continue // ignoramos NFTs (ERC-721/1155)
		}
		bal, ok := new(big.Int).SetString(r.Balance, 10)
		if !ok || bal.Sign() <= 0 {
			continue // sin saldo: fuera
		}
		dec, err := strconv.ParseUint(r.Decimals, 10, 8)
		if err != nil {
			continue // sin decimales no podemos formatear la cantidad
		}
		out = append(out, TokenBalance{
			Token:    common.HexToAddress(r.ContractAddress),
			Symbol:   strings.TrimSpace(r.Symbol),
			Decimals: uint8(dec),
			Balance:  bal,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Symbol < out[j].Symbol })
	if len(out) > maxTokensPerCell {
		out = out[:maxTokensPerCell]
	}
	return out, nil
}
