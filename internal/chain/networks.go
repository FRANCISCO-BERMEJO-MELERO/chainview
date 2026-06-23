// Package chain encapsula todo el acceso a las redes EVM: definición de redes,
// cliente RPC multi-red y (en semanas posteriores) lectura de balances y txs.
package chain

// Chain IDs (EIP-155) de las redes soportadas por defecto. Se exponen como
// constantes para evitar "números mágicos" repartidos por el código.
const (
	ChainEthereum uint64 = 1
	ChainOptimism uint64 = 10
	ChainPolygon  uint64 = 137
	ChainBase     uint64 = 8453
	ChainArbitrum uint64 = 42161
	ChainLinea    uint64 = 59144
	ChainScroll   uint64 = 534352
)

// Network describe una red EVM. Lleva todo lo necesario para operar la red sin
// tablas hardcodeadas paralelas: RPC, explorador, API de Blockscout (txs/tokens)
// y los identificadores de precio de DefiLlama (1.1). Una red definida en el TOML
// es así autosuficiente.
type Network struct {
	Key      string // slug estable en minúsculas, p.ej. "ethereum" (clave de config)
	Name     string // nombre legible, p.ej. "Ethereum"
	ChainID  uint64 // identificador EIP-155 de la cadena
	RPCURL   string // endpoint JSON-RPC por defecto
	Symbol   string // símbolo de la moneda nativa (ETH, POL, …)
	Explorer string // base del explorador de bloques (para enlaces futuros)

	// BlockscoutAPI es la base del API Blockscout (estilo Etherscan) de esta red,
	// usada para historial de txs y descubrimiento de tokens (1.2). Vacío = la red
	// no ofrece esos datos (solo balances nativos y precios).
	BlockscoutAPI string
	// PriceChain es el slug de cadena de DefiLlama para tasar tokens por dirección
	// (`{PriceChain}:{address}`), p.ej. "ethereum", "polygon".
	PriceChain string
	// NativeCoinID es el id de CoinGecko del activo nativo, para tasarlo vía
	// DefiLlama (`coingecko:{NativeCoinID}`), p.ej. "ethereum", "matic-network".
	NativeCoinID string
}

// DefaultNetworks devuelve las redes por defecto en orden de presentación. Las
// cuatro primeras son las históricas (todas ETH); luego un default corto de redes
// extra (Polygon, Scroll, Linea). El resto de redes EVM se añaden vía `[[network]]`
// en el TOML (ver config y config.example.toml).
//
// RPCs públicos de publicnode.com: gratis y SIN API key, a cambio de estar
// rate-limited (lo mitiga la caché/backoff de cache.go).
func DefaultNetworks() []Network {
	return []Network{
		{
			Key:           "ethereum",
			Name:          "Ethereum",
			ChainID:       ChainEthereum,
			RPCURL:        "https://ethereum-rpc.publicnode.com",
			Symbol:        "ETH",
			Explorer:      "https://etherscan.io",
			BlockscoutAPI: "https://eth.blockscout.com/api",
			PriceChain:    "ethereum",
			NativeCoinID:  "ethereum",
		},
		{
			Key:           "arbitrum",
			Name:          "Arbitrum One",
			ChainID:       ChainArbitrum,
			RPCURL:        "https://arbitrum-one-rpc.publicnode.com",
			Symbol:        "ETH",
			Explorer:      "https://arbiscan.io",
			BlockscoutAPI: "https://arbitrum.blockscout.com/api",
			PriceChain:    "arbitrum",
			NativeCoinID:  "ethereum",
		},
		{
			Key:           "base",
			Name:          "Base",
			ChainID:       ChainBase,
			RPCURL:        "https://base-rpc.publicnode.com",
			Symbol:        "ETH",
			Explorer:      "https://basescan.org",
			BlockscoutAPI: "https://base.blockscout.com/api",
			PriceChain:    "base",
			NativeCoinID:  "ethereum",
		},
		{
			Key:           "optimism",
			Name:          "Optimism",
			ChainID:       ChainOptimism,
			RPCURL:        "https://optimism-rpc.publicnode.com",
			Symbol:        "ETH",
			Explorer:      "https://optimistic.etherscan.io",
			BlockscoutAPI: "https://explorer.optimism.io/api",
			PriceChain:    "optimism",
			NativeCoinID:  "ethereum",
		},
		{
			Key:           "polygon",
			Name:          "Polygon",
			ChainID:       ChainPolygon,
			RPCURL:        "https://polygon-bor-rpc.publicnode.com",
			Symbol:        "POL",
			Explorer:      "https://polygonscan.com",
			BlockscoutAPI: "https://polygon.blockscout.com/api",
			PriceChain:    "polygon",
			NativeCoinID:  "polygon-ecosystem-token",
		},
		{
			Key:           "scroll",
			Name:          "Scroll",
			ChainID:       ChainScroll,
			RPCURL:        "https://scroll-rpc.publicnode.com",
			Symbol:        "ETH",
			Explorer:      "https://scrollscan.com",
			BlockscoutAPI: "https://scroll.blockscout.com/api",
			PriceChain:    "scroll",
			NativeCoinID:  "ethereum",
		},
		{
			Key:           "linea",
			Name:          "Linea",
			ChainID:       ChainLinea,
			RPCURL:        "https://linea-rpc.publicnode.com",
			Symbol:        "ETH",
			Explorer:      "https://lineascan.build",
			BlockscoutAPI: "https://explorer.linea.build/api",
			PriceChain:    "linea",
			NativeCoinID:  "ethereum",
		},
	}
}
