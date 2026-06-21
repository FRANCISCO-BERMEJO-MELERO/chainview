package chain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// etherscanV2URL es el endpoint unificado de Etherscan V2: el mismo host sirve
// todas las redes cambiando el parámetro chainid.
const etherscanV2URL = "https://api.etherscan.io/v2/api"

// Tx es una transacción normalizada para mostrar en la TUI.
type Tx struct {
	ChainID     uint64 // red en la que ocurrió (la rellena el provider)
	Hash        string
	From        common.Address
	To          common.Address // dirección cero si es creación de contrato
	Value       *big.Int       // valor en wei
	Input       []byte         // calldata; vacío en una transferencia nativa de ETH
	BlockNumber uint64
	GasUsed     uint64
	GasPrice    *big.Int // precio de gas en wei
	Timestamp   time.Time
	Success     bool
	Nonce       uint64
}

// TxProvider abstrae la fuente del historial de transacciones para poder
// mockearla en tests (la UI depende de la interfaz, no de Etherscan).
type TxProvider interface {
	// RecentTxs devuelve la página `page` (1-based) de `perPage` txs de una
	// dirección en una red, de la más reciente a la más antigua.
	RecentTxs(ctx context.Context, chainID uint64, addr common.Address, page, perPage int) ([]Tx, error)
}

// EtherscanProvider implementa TxProvider contra la API V2 de Etherscan.
type EtherscanProvider struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// NewEtherscanProvider crea el proveedor con la API key dada.
func NewEtherscanProvider(apiKey string) *EtherscanProvider {
	return &EtherscanProvider{
		apiKey:  apiKey,
		baseURL: etherscanV2URL,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

// RecentTxs consulta el historial vía Etherscan V2. Devuelve un error legible si
// falta la API key (en vez de hacer una petición que fallaría).
func (p *EtherscanProvider) RecentTxs(ctx context.Context, chainID uint64, addr common.Address, page, perPage int) ([]Tx, error) {
	if p.apiKey == "" {
		return nil, errors.New("falta la API key de Etherscan (config etherscan_api_key o variable ETHERSCAN_API_KEY)")
	}

	q := url.Values{}
	q.Set("chainid", strconv.FormatUint(chainID, 10))
	q.Set("module", "account")
	q.Set("action", "txlist")
	q.Set("address", addr.Hex())
	q.Set("startblock", "0")
	q.Set("endblock", "99999999")
	q.Set("page", strconv.Itoa(page))
	q.Set("offset", strconv.Itoa(perPage))
	q.Set("sort", "desc") // más recientes primero
	q.Set("apikey", p.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("construyendo petición a Etherscan: %w", err)
	}

	resp, err := p.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("consultando Etherscan: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("leyendo respuesta de Etherscan: %w", err)
	}
	return parseTxList(body, chainID)
}

// etherscanResp es el sobre estándar de Etherscan. Result se deja crudo porque
// es una lista de txs cuando hay éxito, pero un string de error cuando falla.
type etherscanResp struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

// etherscanTx es una tx tal cual la devuelve la API (todo en strings).
type etherscanTx struct {
	Hash            string `json:"hash"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           string `json:"value"`
	Input           string `json:"input"`
	BlockNumber     string `json:"blockNumber"`
	GasUsed         string `json:"gasUsed"`
	GasPrice        string `json:"gasPrice"`
	TimeStamp       string `json:"timeStamp"`
	IsError         string `json:"isError"`
	TxReceiptStatus string `json:"txreceipt_status"`
	Nonce           string `json:"nonce"`
}

// parseTxList traduce el cuerpo JSON del formato Etherscan a []Tx, etiquetando
// cada tx con su `chainID`. Separada del HTTP para poder testear el parseo con
// fixtures sin red, y compartida por ambos proveedores: Blockscout expone una API
// compatible con la de Etherscan, así que devuelve el mismo sobre
// (status/message/result).
func parseTxList(body []byte, chainID uint64) ([]Tx, error) {
	var resp etherscanResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("respuesta de Etherscan ilegible: %w", err)
	}

	if resp.Status != "1" {
		// status "0" puede ser "sin transacciones" (lista vacía, no es error)...
		if strings.Contains(strings.ToLower(resp.Message), "no transactions found") {
			return []Tx{}, nil
		}
		// ...o un error real (key inválida, rate-limit), con Result como string.
		var detail string
		_ = json.Unmarshal(resp.Result, &detail)
		if detail == "" {
			detail = resp.Message
		}
		return nil, fmt.Errorf("Etherscan: %s", detail)
	}

	var raw []etherscanTx
	if err := json.Unmarshal(resp.Result, &raw); err != nil {
		return nil, fmt.Errorf("lista de txs ilegible: %w", err)
	}

	txs := make([]Tx, 0, len(raw))
	for _, r := range raw {
		tx := r.toTx()
		tx.ChainID = chainID
		txs = append(txs, tx)
	}
	return txs, nil
}

// toTx convierte la tx en strings de Etherscan a la struct interna.
func (e etherscanTx) toTx() Tx {
	value, ok := new(big.Int).SetString(e.Value, 10)
	if !ok {
		value = big.NewInt(0)
	}
	ts, _ := strconv.ParseInt(e.TimeStamp, 10, 64)
	nonce, _ := strconv.ParseUint(e.Nonce, 10, 64)
	block, _ := strconv.ParseUint(e.BlockNumber, 10, 64)
	gasUsed, _ := strconv.ParseUint(e.GasUsed, 10, 64)
	gasPrice, ok := new(big.Int).SetString(e.GasPrice, 10)
	if !ok {
		gasPrice = big.NewInt(0)
	}

	// Una tx tuvo éxito si no marcó error y el receipt es "1". Las txs antiguas
	// (pre-Byzantium) no traen txreceipt_status: las tratamos como exitosas si
	// isError es "0".
	success := e.IsError == "0" && (e.TxReceiptStatus == "1" || e.TxReceiptStatus == "")

	return Tx{
		Hash:        e.Hash,
		From:        common.HexToAddress(e.From),
		To:          common.HexToAddress(e.To),
		Value:       value,
		Input:       common.FromHex(e.Input),
		BlockNumber: block,
		GasUsed:     gasUsed,
		GasPrice:    gasPrice,
		Timestamp:   time.Unix(ts, 0),
		Success:     success,
		Nonce:       nonce,
	}
}
