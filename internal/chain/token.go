package chain

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// TokenMeta son los metadatos de un token necesarios para formatear cantidades
// de forma legible: su símbolo y sus decimales (USDC usa 6, no 18).
type TokenMeta struct {
	Symbol   string
	Decimals uint8
}

// TokenResolver traduce la dirección de un contrato de token a sus metadatos.
// La UI depende de esta interfaz (no de *Client) para poder mockearla en tests.
// El segundo valor es false si no se pudo resolver (token desconocido o sin RPC).
type TokenResolver interface {
	TokenMeta(ctx context.Context, chainID uint64, token common.Address) (TokenMeta, bool)
}

// knownTokens es la tabla local de tokens populares por chain. Es el camino
// rápido (sin RPC) para el grueso de las transferencias reales; lo que no esté
// aquí cae al fallback por RPC en Client.TokenMeta. Las direcciones se normalizan
// con HexToAddress, así que el checksum del literal es indiferente.
var knownTokens = map[uint64]map[common.Address]TokenMeta{
	ChainEthereum: {
		common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"): {"USDC", 6},
		common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"): {"USDT", 6},
		common.HexToAddress("0x6B175474E89094C44Da98b954EedeAC495271d0F"): {"DAI", 18},
		common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"): {"WETH", 18},
	},
	ChainArbitrum: {
		common.HexToAddress("0xaf88d065e77c8cC2239327C5EDb3A432268e5831"): {"USDC", 6},
		common.HexToAddress("0xFd086bC7CD5C481DCC9C85ebE478A1C0b69FCbb9"): {"USDT", 6},
		common.HexToAddress("0x82aF49447D8a07e3bd95BD0d56f35241523fBab1"): {"WETH", 18},
	},
	ChainBase: {
		common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"): {"USDC", 6},
		common.HexToAddress("0x4200000000000000000000000000000000000006"): {"WETH", 18},
	},
	ChainOptimism: {
		common.HexToAddress("0x0b2C639c533813f4Aa9D7837CAf62653d097Ff85"): {"USDC", 6},
		common.HexToAddress("0x4200000000000000000000000000000000000006"): {"WETH", 18},
	},
}

// metaABI describe los getters ERC-20 que consultamos por RPC para los tokens
// fuera de la tabla local.
const metaABIJSON = `[
	{"name":"symbol","type":"function","stateMutability":"view","inputs":[],"outputs":[{"type":"string"}]},
	{"name":"decimals","type":"function","stateMutability":"view","inputs":[],"outputs":[{"type":"uint8"}]}
]`

var metaABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(metaABIJSON))
	if err != nil {
		panic("chain: ABI de metadatos inválida: " + err.Error())
	}
	return a
}()

// TokenMeta resuelve los metadatos de un token en tres pasos: (1) tabla local de
// tokens conocidos, (2) caché en memoria, (3) consulta RPC a symbol()/decimals().
// Cachea también los fallos (entrada con ok=false) para no repetir RPC ante un
// token que no implementa el estándar.
func (c *Client) TokenMeta(ctx context.Context, chainID uint64, token common.Address) (TokenMeta, bool) {
	// (1) Tabla local: camino rápido sin red.
	if byChain, ok := knownTokens[chainID]; ok {
		if meta, ok := byChain[token]; ok {
			return meta, true
		}
	}

	// (2) Caché (incluye negativos).
	key := fmt.Sprintf("%d:%s", chainID, token.Hex())
	c.tokenMu.RLock()
	entry, cached := c.tokenCache[key]
	c.tokenMu.RUnlock()
	if cached {
		return entry.meta, entry.ok
	}

	// (3) Fallback por RPC. El resultado (éxito o fallo) se cachea siempre.
	meta, ok := c.fetchTokenMeta(ctx, chainID, token)
	c.tokenMu.Lock()
	c.tokenCache[key] = tokenEntry{meta: meta, ok: ok}
	c.tokenMu.Unlock()
	return meta, ok
}

// fetchTokenMeta consulta symbol() y decimals() del contrato vía eth_call. Si
// cualquiera de las dos falla, consideramos el token no resoluble (ok=false): sin
// decimales no podemos formatear la cantidad con sentido.
func (c *Client) fetchTokenMeta(ctx context.Context, chainID uint64, token common.Address) (TokenMeta, bool) {
	conn, err := c.connect(chainID)
	if err != nil {
		return TokenMeta{}, false
	}

	symbol, err := callSymbol(ctx, conn, token)
	if err != nil {
		return TokenMeta{}, false
	}
	decimals, err := callDecimals(ctx, conn, token)
	if err != nil {
		return TokenMeta{}, false
	}
	return TokenMeta{Symbol: symbol, Decimals: decimals}, true
}

// contractCaller es el subconjunto de ethclient que usamos para los eth_call de
// metadatos. Tenerlo como interfaz mantiene fetchTokenMeta testeable.
type contractCaller interface {
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

func callDecimals(ctx context.Context, caller contractCaller, token common.Address) (uint8, error) {
	out, err := ethCall(ctx, caller, token, "decimals")
	if err != nil {
		return 0, err
	}
	var dec uint8
	if err := metaABI.UnpackIntoInterface(&dec, "decimals", out); err != nil {
		return 0, err
	}
	return dec, nil
}

// callSymbol lee symbol(). Algunos tokens antiguos (p.ej. MKR) devuelven bytes32
// en vez de string; lo intentamos como string y, si falla, interpretamos los
// bytes crudos recortando los ceros de padding.
func callSymbol(ctx context.Context, caller contractCaller, token common.Address) (string, error) {
	out, err := ethCall(ctx, caller, token, "symbol")
	if err != nil {
		return "", err
	}
	var symbol string
	if err := metaABI.UnpackIntoInterface(&symbol, "symbol", out); err == nil {
		return symbol, nil
	}
	// Fallback bytes32: descartamos los \x00 de relleno.
	s := strings.TrimRight(string(out), "\x00")
	if s == "" {
		return "", fmt.Errorf("symbol() ilegible para %s", token.Hex())
	}
	return strings.TrimSpace(s), nil
}

// ethCall empaqueta la llamada al método (sin argumentos) y la ejecuta.
func ethCall(ctx context.Context, caller contractCaller, token common.Address, method string) ([]byte, error) {
	data, err := metaABI.Pack(method)
	if err != nil {
		return nil, err
	}
	return caller.CallContract(ctx, ethereum.CallMsg{To: &token, Data: data}, nil)
}
