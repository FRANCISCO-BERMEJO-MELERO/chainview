package chain

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// FuzzParseTxList comprueba que parsear cuerpos arbitrarios de la API de
// Etherscan/Blockscout nunca entra en pánico, y que cuando devuelve txs estas son
// coherentes (chainID propagado, Value no-nil). La propiedad es la robustez, no la
// igualdad: la mayoría de entradas serán JSON inválido (error limpio) y está bien.
func FuzzParseTxList(f *testing.F) {
	f.Add(txListFixture)
	f.Add(`{"status":"0","message":"No transactions found","result":[]}`)
	f.Add(`{"status":"0","message":"NOTOK","result":"Invalid API Key"}`)
	f.Add(`{"status":"1","message":"OK","result":[{"value":"x","nonce":"y"}]}`)
	f.Add(``)

	const chainID = ChainEthereum
	f.Fuzz(func(t *testing.T, body string) {
		txs, err := parseTxList([]byte(body), chainID)
		if err != nil {
			return // error limpio: la entrada no era una respuesta válida
		}
		for i, tx := range txs {
			if tx.ChainID != chainID {
				t.Fatalf("tx[%d].ChainID = %d, esperaba %d", i, tx.ChainID, chainID)
			}
			if tx.Value == nil {
				t.Fatalf("tx[%d].Value es nil; toTx debe garantizar un big.Int", i)
			}
			if tx.GasPrice == nil {
				t.Fatalf("tx[%d].GasPrice es nil; toTx debe garantizar un big.Int", i)
			}
		}
	})
}

// FuzzDecodeCall comprueba que decodificar calldata arbitrario no entra en pánico
// (índices de slice, type assertions del Unpack) y que un resultado "ok" es
// internamente coherente.
func FuzzDecodeCall(f *testing.F) {
	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	from := common.HexToAddress("0x1111111111111111111111111111111111111111")
	if data, err := tokenABI.Pack("transfer", to, big.NewInt(1000)); err == nil {
		f.Add(data)
	}
	if data, err := tokenABI.Pack("transferFrom", from, to, big.NewInt(1)); err == nil {
		f.Add(data)
	}
	if data, err := tokenABI.Pack("approve", to, big.NewInt(0)); err == nil {
		f.Add(data)
	}
	f.Add([]byte{})                       // vacío
	f.Add([]byte{0x01, 0x02})             // < 4 bytes
	f.Add([]byte{0xa9, 0x05, 0x9c, 0xbb}) // selector de transfer sin argumentos

	f.Fuzz(func(t *testing.T, input []byte) {
		call, ok := DecodeCall(input)
		if !ok {
			return
		}
		// Un acierto tiene un Kind conocido y un Value no-nil (cantidad o tokenId).
		switch call.Kind {
		case CallTransfer, CallTransferFrom, CallApprove:
		default:
			t.Fatalf("DecodeCall ok con Kind inválido: %d", call.Kind)
		}
		if call.Value == nil {
			t.Fatal("DecodeCall ok con Value nil")
		}
	})
}

// FuzzFormatUnits comprueba que el formateo entero de valores nunca entra en
// pánico ni deja un separador decimal colgante. Se clampan los decimales a un
// rango sano: FormatUnits hace 10^decimals, así que un valor enorme del fuzzer
// dispararía un Exp inviable (no es un bug del parser).
func FuzzFormatUnits(f *testing.F) {
	f.Add("0", 18)
	f.Add("1000000000000000000", 18)
	f.Add("1", 18)
	f.Add("123456", 6)
	f.Add("-5", 0)

	f.Fuzz(func(t *testing.T, valueStr string, decimals int) {
		value, ok := new(big.Int).SetString(valueStr, 10)
		if !ok {
			return // no era un entero base 10
		}
		if decimals < 0 || decimals > 64 {
			return // fuera del rango razonable de decimales de un token
		}

		out := FormatUnits(value, decimals)
		if out == "" {
			t.Fatalf("FormatUnits(%s, %d) devolvió cadena vacía", valueStr, decimals)
		}
		if strings.HasSuffix(out, ".") {
			t.Fatalf("FormatUnits(%s, %d) = %q deja un punto colgante", valueStr, decimals, out)
		}
		if i := strings.IndexByte(out, '.'); i >= 0 && strings.HasSuffix(out, "0") {
			t.Fatalf("FormatUnits(%s, %d) = %q deja un cero final en la parte decimal", valueStr, decimals, out)
		}
	})
}
