// Package chain encapsula todo el acceso a las redes EVM: definición de redes,
// cliente RPC multi-red y (en semanas posteriores) lectura de balances y txs.
package chain

// Chain IDs (EIP-155) de las redes soportadas por defecto. Se exponen como
// constantes para evitar "números mágicos" repartidos por el código.
const (
	ChainEthereum uint64 = 1
	ChainOptimism uint64 = 10
	ChainBase     uint64 = 8453
	ChainArbitrum uint64 = 42161
)

// Network describe una red EVM. El RPCURL es el endpoint por defecto; en la
// Semana 4 la config TOML podrá sobreescribirlo por red (p.ej. para usar
// Alchemy/Infura en vez del RPC público).
type Network struct {
	Name     string // nombre legible, p.ej. "Ethereum"
	ChainID  uint64 // identificador EIP-155 de la cadena
	RPCURL   string // endpoint JSON-RPC por defecto
	Symbol   string // símbolo de la moneda nativa (todas usan ETH aquí)
	Explorer string // base del explorador de bloques (para enlaces futuros)
}

// DefaultNetworks devuelve las 4 redes por defecto en orden de presentación.
// RPCs públicos de publicnode.com: gratis y sin API key, a cambio de estar
// rate-limited (lo mitigaremos con caché/backoff en la Semana 9).
func DefaultNetworks() []Network {
	return []Network{
		{
			Name:     "Ethereum",
			ChainID:  ChainEthereum,
			RPCURL:   "https://ethereum-rpc.publicnode.com",
			Symbol:   "ETH",
			Explorer: "https://etherscan.io",
		},
		{
			Name:     "Arbitrum One",
			ChainID:  ChainArbitrum,
			RPCURL:   "https://arbitrum-one-rpc.publicnode.com",
			Symbol:   "ETH",
			Explorer: "https://arbiscan.io",
		},
		{
			Name:     "Base",
			ChainID:  ChainBase,
			RPCURL:   "https://base-rpc.publicnode.com",
			Symbol:   "ETH",
			Explorer: "https://basescan.org",
		},
		{
			Name:     "Optimism",
			ChainID:  ChainOptimism,
			RPCURL:   "https://optimism-rpc.publicnode.com",
			Symbol:   "ETH",
			Explorer: "https://optimistic.etherscan.io",
		},
	}
}
