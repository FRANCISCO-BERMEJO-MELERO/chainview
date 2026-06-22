// Package config carga la configuración de chainview desde un archivo TOML
// ubicado vía XDG. Si no existe, se usan valores por defecto: la app arranca
// sin que el usuario tenga que configurar nada.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
)

// configPath es la ruta relativa dentro del directorio de config XDG
// (p.ej. ~/.config/chainview/config.toml en Linux).
const configPath = "chainview/config.toml"

// defaultRefreshSeconds es el intervalo de refresco de balances por defecto
// (decisión D2: polling cada ~15 s).
const defaultRefreshSeconds = 15

// Config es la configuración efectiva de la app.
type Config struct {
	// RefreshSeconds es cada cuántos segundos se refrescan los balances.
	RefreshSeconds int `toml:"refresh_seconds"`
	// EtherscanAPIKey es OPCIONAL. Por defecto el historial de txs se obtiene de
	// Blockscout sin ninguna key; si se define esta key (en el TOML o en la
	// variable de entorno ETHERSCAN_API_KEY, que tiene prioridad), se usa
	// Etherscan en su lugar.
	EtherscanAPIKey string `toml:"etherscan_api_key"`
	// RPC mapea el slug de una red (chain.Network.Key, p.ej. "ethereum") a una
	// URL RPC que sobreescribe la pública por defecto. Permite usar Alchemy/Infura.
	// Vía heredada; `[[network]]` es la forma completa y recomendada.
	RPC map[string]string `toml:"rpc"`
	// Network son redes definidas/sobreescritas por el usuario (1.4). Si el
	// chain_id coincide con una del catálogo, la sobreescribe (solo los campos no
	// vacíos); si no, la añade.
	Network []NetworkConfig `toml:"network"`
}

// NetworkConfig es una red declarada en el TOML (bloque `[[network]]`). Los
// campos vacíos no sobreescriben a la red base correspondiente.
type NetworkConfig struct {
	Key           string `toml:"key"`
	Name          string `toml:"name"`
	ChainID       uint64 `toml:"chain_id"`
	RPCURL        string `toml:"rpc_url"`
	Symbol        string `toml:"symbol"`
	Explorer      string `toml:"explorer"`
	BlockscoutAPI string `toml:"blockscout_api"`
	PriceChain    string `toml:"price_chain"`
	NativeCoinID  string `toml:"native_coin_id"`
}

// Default devuelve la configuración por defecto (sin overrides de RPC ni redes).
func Default() Config {
	return Config{
		RefreshSeconds: defaultRefreshSeconds,
		RPC:            map[string]string{},
	}
}

// Load lee la config desde el archivo TOML en el directorio XDG. Si el archivo
// no existe, devuelve los valores por defecto sin error (caso normal en el
// primer arranque). Solo devuelve error si el archivo existe pero está corrupto.
func Load() (Config, error) {
	cfg := Default()

	path, err := xdg.ConfigFile(configPath)
	if err != nil {
		return cfg, fmt.Errorf("resolviendo ruta de config: %w", err)
	}
	return loadFrom(path)
}

// loadFrom carga la config desde una ruta concreta. Separada de Load para poder
// testearla con un archivo temporal sin depender del entorno XDG real.
func loadFrom(path string) (Config, error) {
	cfg := Default()

	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return cfg, nil // sin archivo: defaults
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Default(), fmt.Errorf("leyendo config %s: %w", path, err)
	}

	// Saneamos valores inválidos en lugar de fallar.
	if cfg.RefreshSeconds <= 0 {
		cfg.RefreshSeconds = defaultRefreshSeconds
	}
	if cfg.RPC == nil {
		cfg.RPC = map[string]string{}
	}
	return cfg, nil
}

// Networks devuelve el catálogo efectivo: las redes por defecto, los overrides de
// RPC del bloque `[rpc]` (vía heredada) y, encima, las redes del bloque
// `[[network]]` (que sobreescriben por chain_id o se añaden). Las entradas de
// `[[network]]` inválidas (sin chain_id, o nuevas sin rpc_url) se descartan sin
// romper la carga.
func (c Config) Networks() []chain.Network {
	nets := chain.DefaultNetworks()

	// (1) Overrides de RPC por Key (compatibilidad hacia atrás).
	for i := range nets {
		if url, ok := c.RPC[nets[i].Key]; ok && url != "" {
			nets[i].RPCURL = url
		}
	}

	// (2) Bloque [[network]]: override por chain_id o alta.
	idx := make(map[uint64]int, len(nets))
	for i, n := range nets {
		idx[n.ChainID] = i
	}
	for _, nc := range c.Network {
		if nc.ChainID == 0 {
			continue // sin chain_id no podemos identificar ni dar de alta la red
		}
		if i, ok := idx[nc.ChainID]; ok {
			nets[i] = overlayNetwork(nets[i], nc) // override de campos no vacíos
			continue
		}
		if nc.RPCURL == "" {
			continue // una red nueva necesita al menos un RPC
		}
		nets = append(nets, overlayNetwork(chain.Network{ChainID: nc.ChainID}, nc))
		idx[nc.ChainID] = len(nets) - 1
	}
	return nets
}

// overlayNetwork copia sobre base solo los campos no vacíos de nc, de modo que un
// `[[network]]` puede tocar únicamente lo que quiera cambiar (p.ej. solo el RPC)
// sin perder los metadatos por defecto de esa red.
func overlayNetwork(base chain.Network, nc NetworkConfig) chain.Network {
	if nc.Key != "" {
		base.Key = nc.Key
	}
	if nc.Name != "" {
		base.Name = nc.Name
	}
	if nc.RPCURL != "" {
		base.RPCURL = nc.RPCURL
	}
	if nc.Symbol != "" {
		base.Symbol = nc.Symbol
	}
	if nc.Explorer != "" {
		base.Explorer = nc.Explorer
	}
	if nc.BlockscoutAPI != "" {
		base.BlockscoutAPI = nc.BlockscoutAPI
	}
	if nc.PriceChain != "" {
		base.PriceChain = nc.PriceChain
	}
	if nc.NativeCoinID != "" {
		base.NativeCoinID = nc.NativeCoinID
	}
	return base
}
