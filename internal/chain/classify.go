package chain

import "github.com/ethereum/go-ethereum/common"

// TxKind clasifica una tx desde el punto de vista de una wallet concreta, al
// estilo de las etiquetas de Etherscan (entrante / saliente / llamada a
// contrato / despliegue). Es una clasificación de presentación: el detalle fino
// de qué hace la tx lo da el decodificador (DecodeCall) y la descripción.
type TxKind int

const (
	KindUnknown TxKind = iota
	KindIn             // recibida: la wallet es el destino de un envío nativo
	KindOut            // enviada: la wallet origina un envío nativo
	KindSelf           // envío nativo de la wallet a sí misma
	KindCall           // interacción con contrato (hay calldata)
	KindNew            // despliegue de contrato (sin destinatario)
)

// ClassifyTx deduce el tipo de una tx respecto a `wallet`. La precedencia es:
// despliegue > llamada a contrato > self > saliente > entrante. Así una
// transferencia de token (que es una llamada a contrato) se etiqueta como CALL,
// y solo los envíos nativos sin calldata se etiquetan IN/OUT/SELF.
func ClassifyTx(tx Tx, wallet common.Address) TxKind {
	if tx.To == (common.Address{}) {
		return KindNew
	}
	if len(tx.Input) >= 4 {
		return KindCall
	}
	switch {
	case tx.From == wallet && tx.To == wallet:
		return KindSelf
	case tx.From == wallet:
		return KindOut
	case tx.To == wallet:
		return KindIn
	default:
		return KindUnknown
	}
}

// String devuelve una etiqueta corta y estable (útil para exportar a CSV).
func (k TxKind) String() string {
	switch k {
	case KindIn:
		return "IN"
	case KindOut:
		return "OUT"
	case KindSelf:
		return "SELF"
	case KindCall:
		return "CALL"
	case KindNew:
		return "NEW"
	default:
		return ""
	}
}
