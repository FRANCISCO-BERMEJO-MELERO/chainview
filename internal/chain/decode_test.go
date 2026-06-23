package chain

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// pack genera el calldata real de un método usando la misma ABI que decodifica
// el código de producción. Así el test no hardcodea selectores: comprueba el
// round-trip pack -> DecodeCall.
func pack(t *testing.T, method string, args ...interface{}) []byte {
	t.Helper()
	data, err := tokenABI.Pack(method, args...)
	if err != nil {
		t.Fatalf("empaquetando %s: %v", method, err)
	}
	return data
}

func TestDecodeCall(t *testing.T) {
	to := common.HexToAddress("0x1111111111111111111111111111111111111111")
	from := common.HexToAddress("0x2222222222222222222222222222222222222222")
	value := big.NewInt(100_000000) // 100 USDC (6 decimales)

	tests := []struct {
		name     string
		input    []byte
		wantKind CallKind
		wantFrom common.Address
		wantTo   common.Address
		wantVal  *big.Int
	}{
		{
			name:     "transfer",
			input:    pack(t, "transfer", to, value),
			wantKind: CallTransfer,
			wantTo:   to,
			wantVal:  value,
		},
		{
			name:     "approve",
			input:    pack(t, "approve", to, value),
			wantKind: CallApprove,
			wantTo:   to,
			wantVal:  value,
		},
		{
			name:     "transferFrom",
			input:    pack(t, "transferFrom", from, to, value),
			wantKind: CallTransferFrom,
			wantFrom: from,
			wantTo:   to,
			wantVal:  value,
		},
		{
			name:     "safeTransferFrom",
			input:    pack(t, "safeTransferFrom", from, to, big.NewInt(42)),
			wantKind: CallTransferFrom,
			wantFrom: from,
			wantTo:   to,
			wantVal:  big.NewInt(42),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := DecodeCall(tt.input)
			if !ok {
				t.Fatalf("DecodeCall devolvió ok=false para %s", tt.name)
			}
			if got.Kind != tt.wantKind {
				t.Errorf("Kind = %d, quiero %d", got.Kind, tt.wantKind)
			}
			if got.From != tt.wantFrom {
				t.Errorf("From = %s, quiero %s", got.From, tt.wantFrom)
			}
			if got.To != tt.wantTo {
				t.Errorf("To = %s, quiero %s", got.To, tt.wantTo)
			}
			if got.Value.Cmp(tt.wantVal) != 0 {
				t.Errorf("Value = %s, quiero %s", got.Value, tt.wantVal)
			}
		})
	}
}

func TestDecodeCallRejectsUnknown(t *testing.T) {
	cases := map[string][]byte{
		"vacío (transferencia nativa)": nil,
		"demasiado corto":              {0xa9, 0x05},
		"selector desconocido":         {0xde, 0xad, 0xbe, 0xef},
		"transfer truncado":            append([]byte{0xa9, 0x05, 0x9c, 0xbb}, 0x01), // selector ok, args corruptos
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, ok := DecodeCall(input); ok {
				t.Errorf("DecodeCall(%x) = ok, quiero false", input)
			}
		})
	}
}
