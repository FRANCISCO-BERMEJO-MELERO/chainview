package chain

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// CallKind clasifica el tipo de llamada a contrato que sabemos reconocer. Los
// selectores de estos métodos son idénticos en ERC-20 y ERC-721 (la firma es la
// misma); por eso aquí solo identificamos la *acción*, y es la resolución de
// metadatos del token (símbolo/decimales) la que decide cómo formatear el valor.
type CallKind int

const (
	CallUnknown      CallKind = iota
	CallTransfer              // transfer(to, value)
	CallTransferFrom          // transferFrom / safeTransferFrom (from, to, value)
	CallApprove               // approve(spender, value)
)

// DecodedCall es el resultado de decodificar el calldata de una tx contra los
// métodos conocidos de token. Value es el argumento crudo (sin aplicar
// decimales): una cantidad en ERC-20 o un tokenId en ERC-721.
type DecodedCall struct {
	Kind  CallKind
	From  common.Address // solo en transferFrom/safeTransferFrom
	To    common.Address // destinatario (transfer/transferFrom) o spender (approve)
	Value *big.Int
}

// tokenABI describe los métodos de token que sabemos decodificar. Mantenerlo como
// una ABI declarativa (en vez de pelear con los selectores a mano) nos deja
// reusar el desempaquetado de argumentos de go-ethereum, que ya valida tamaños.
const tokenABIJSON = `[
	{"name":"transfer","type":"function","inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}]},
	{"name":"transferFrom","type":"function","inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"value","type":"uint256"}]},
	{"name":"safeTransferFrom","type":"function","inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"tokenId","type":"uint256"}]},
	{"name":"approve","type":"function","inputs":[{"name":"spender","type":"address"},{"name":"value","type":"uint256"}]}
]`

// tokenABI se compila una sola vez al cargar el paquete. Si el JSON de arriba
// fuese inválido es un bug del programador, no un error en runtime: por eso
// usamos panic en vez de devolver un error.
var tokenABI = func() abi.ABI {
	a, err := abi.JSON(strings.NewReader(tokenABIJSON))
	if err != nil {
		panic("chain: invalid token ABI: " + err.Error())
	}
	return a
}()

// DecodeCall intenta decodificar el calldata de una tx como una llamada de token
// conocida. El segundo valor es false si el calldata está vacío, es demasiado
// corto, o el selector/argumentos no corresponden a un método que sepamos leer.
func DecodeCall(input []byte) (DecodedCall, bool) {
	// Las 4 primeras bytes son el selector; sin ellas no hay nada que decodificar
	// (una transferencia nativa de ETH lleva el calldata vacío).
	if len(input) < 4 {
		return DecodedCall{}, false
	}

	method, err := tokenABI.MethodById(input[:4])
	if err != nil {
		return DecodedCall{}, false // selector desconocido
	}

	args, err := method.Inputs.Unpack(input[4:])
	if err != nil {
		return DecodedCall{}, false // argumentos malformados
	}

	switch method.Name {
	case "transfer":
		return DecodedCall{
			Kind:  CallTransfer,
			To:    args[0].(common.Address),
			Value: args[1].(*big.Int),
		}, true
	case "approve":
		return DecodedCall{
			Kind:  CallApprove,
			To:    args[0].(common.Address),
			Value: args[1].(*big.Int),
		}, true
	case "transferFrom", "safeTransferFrom":
		return DecodedCall{
			Kind:  CallTransferFrom,
			From:  args[0].(common.Address),
			To:    args[1].(common.Address),
			Value: args[2].(*big.Int),
		}, true
	}
	return DecodedCall{}, false
}
