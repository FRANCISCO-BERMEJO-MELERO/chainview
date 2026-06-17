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
	// EtherscanAPIKey es la key para el historial de txs (Etherscan V2). Puede
	// sobreescribirse con la variable de entorno ETHERSCAN_API_KEY.
	EtherscanAPIKey string `toml:"etherscan_api_key"`
	// RPC mapea el slug de una red (chain.Network.Key, p.ej. "ethereum") a una
	// URL RPC que sobreescribe la pública por defecto. Permite usar Alchemy/Infura.
	RPC map[string]string `toml:"rpc"`
}

// Default devuelve la configuración por defecto (sin overrides de RPC).
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

// Networks devuelve las redes por defecto con los overrides de RPC aplicados
// según la config (por Network.Key).
func (c Config) Networks() []chain.Network {
	nets := chain.DefaultNetworks()
	for i := range nets {
		if url, ok := c.RPC[nets[i].Key]; ok && url != "" {
			nets[i].RPCURL = url
		}
	}
	return nets
}
