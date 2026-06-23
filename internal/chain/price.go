package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/sync/singleflight"
)

// defiLlamaURL es el endpoint keyless de precios actuales de DefiLlama. Acepta
// varias claves de activo separadas por coma y devuelve sus precios en USD.
const defiLlamaURL = "https://coins.llama.fi/prices/current/"

// priceTTL es la ventana de caché de un precio. Más larga que la de balances
// (rpcTTL): los precios se mueven despacio y la API es pública, así evitamos
// martillearla en cada refresco de 15 s.
const priceTTL = 60 * time.Second

// PriceQuery identifica un activo a tasar: red + token. Token cero = moneda
// nativa de la red.
type PriceQuery struct {
	ChainID uint64
	Token   common.Address // common.Address{} = activo nativo de la red
}

// IsNative indica si la query es para el activo nativo de la red.
func (q PriceQuery) IsNative() bool {
	return q.Token == (common.Address{})
}

// PriceProvider tasa activos en USD. Keyless por defecto. La UI depende de la
// interfaz (no del tipo concreto) para poder mockearla en tests.
type PriceProvider interface {
	// Prices devuelve el precio en USD de cada query resoluble. Las queries sin
	// precio simplemente no aparecen en el mapa (no es error).
	Prices(ctx context.Context, qs []PriceQuery) (map[PriceQuery]float64, error)
}

// priceMeta son los identificadores de precio de una red (de su Network): el slug
// de cadena de DefiLlama para tokens y el id de CoinGecko del activo nativo.
type priceMeta struct {
	priceChain   string
	nativeCoinID string
}

// priceEntry es un precio cacheado con su marca de tiempo. ok=false memoriza que
// el activo no tiene precio, para no reconsultarlo en cada tanda.
type priceEntry struct {
	price float64
	ok    bool
	at    time.Time
}

// DefiLlamaPrices implementa PriceProvider contra coins.llama.fi (keyless, batch).
type DefiLlamaPrices struct {
	baseURL string
	http    *http.Client
	meta    map[uint64]priceMeta // chain ID -> identificadores de precio

	mu    sync.Mutex
	cache map[string]priceEntry // clave de activo DefiLlama -> precio cacheado
	sf    singleflight.Group    // coalescing de tandas idénticas concurrentes
}

// NewDefiLlamaPrices crea el proveedor a partir de las redes efectivas (de donde
// salen los identificadores de precio de cada cadena).
func NewDefiLlamaPrices(networks []Network) *DefiLlamaPrices {
	meta := make(map[uint64]priceMeta, len(networks))
	for _, n := range networks {
		meta[n.ChainID] = priceMeta{priceChain: n.PriceChain, nativeCoinID: n.NativeCoinID}
	}
	return &DefiLlamaPrices{
		baseURL: defiLlamaURL,
		http:    &http.Client{Timeout: 15 * time.Second},
		meta:    meta,
		cache:   map[string]priceEntry{},
	}
}

// coinKey construye la clave de activo de DefiLlama para una query, o "" si la red
// no aporta los identificadores necesarios.
func (p *DefiLlamaPrices) coinKey(q PriceQuery) string {
	m, ok := p.meta[q.ChainID]
	if !ok {
		return ""
	}
	if q.IsNative() {
		if m.nativeCoinID == "" {
			return ""
		}
		return "coingecko:" + m.nativeCoinID
	}
	if m.priceChain == "" {
		return ""
	}
	return m.priceChain + ":" + strings.ToLower(q.Token.Hex())
}

// Prices tasa las queries en USD: sirve de caché lo fresco y pide en una sola
// petición batch las claves que falten.
func (p *DefiLlamaPrices) Prices(ctx context.Context, qs []PriceQuery) (map[PriceQuery]float64, error) {
	now := time.Now()
	out := make(map[PriceQuery]float64, len(qs))

	// Mapeamos cada query a su clave de activo y reunimos las claves a pedir.
	keyOf := make(map[PriceQuery]string, len(qs))
	missing := map[string]struct{}{}
	p.mu.Lock()
	for _, q := range qs {
		key := p.coinKey(q)
		if key == "" {
			continue // red sin identificadores de precio: query no tasable
		}
		keyOf[q] = key
		if e, ok := p.cache[key]; ok && now.Sub(e.at) < priceTTL {
			if e.ok {
				out[q] = e.price
			}
			continue
		}
		missing[key] = struct{}{}
	}
	p.mu.Unlock()

	if len(missing) > 0 {
		if err := p.fetchInto(ctx, missing); err != nil {
			// Ante fallo servimos lo que ya había en caché (out) y propagamos el
			// error para que la UI pueda avisar; nunca dejamos la tabla a oscuras.
			p.fillFromCache(out, keyOf)
			return out, err
		}
	}

	// Con la caché ya repoblada, completamos las queries que faltaban.
	p.fillFromCache(out, keyOf)
	return out, nil
}

// fillFromCache vuelca a out los precios cacheados frescos de las queries dadas.
func (p *DefiLlamaPrices) fillFromCache(out map[PriceQuery]float64, keyOf map[PriceQuery]string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for q, key := range keyOf {
		if _, done := out[q]; done {
			continue
		}
		if e, ok := p.cache[key]; ok && e.ok {
			out[q] = e.price
		}
	}
}

// llamaResp es el sobre de coins.llama.fi: clave de activo -> datos de precio.
type llamaResp struct {
	Coins map[string]struct {
		Price float64 `json:"price"`
	} `json:"coins"`
}

// fetchInto pide las claves indicadas en una sola petición batch y cachea el
// resultado (precios y, como negativos, las claves que la API no devolvió). Usa
// single-flight para coalescer tandas idénticas concurrentes.
func (p *DefiLlamaPrices) fetchInto(ctx context.Context, missing map[string]struct{}) error {
	keys := make([]string, 0, len(missing))
	for k := range missing {
		keys = append(keys, k)
	}
	sort.Strings(keys) // clave de single-flight estable
	sfKey := strings.Join(keys, ",")

	_, err, _ := p.sf.Do(sfKey, func() (interface{}, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+sfKey, nil)
		if err != nil {
			return nil, fmt.Errorf("building DefiLlama request: %w", err)
		}
		resp, err := p.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("querying DefiLlama: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("reading DefiLlama response: %w", err)
		}
		var parsed llamaResp
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, fmt.Errorf("unreadable DefiLlama response: %w", err)
		}

		now := time.Now()
		p.mu.Lock()
		for _, key := range keys {
			c, ok := parsed.Coins[key]
			p.cache[key] = priceEntry{price: c.Price, ok: ok && c.Price > 0, at: now}
		}
		p.mu.Unlock()
		return nil, nil
	})
	return err
}
