package chain

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestClassifyTx(t *testing.T) {
	me := common.HexToAddress("0x1111111111111111111111111111111111111111")
	other := common.HexToAddress("0x2222222222222222222222222222222222222222")
	zero := common.Address{}
	transferInput := []byte{0xa9, 0x05, 0x9c, 0xbb, 0x00} // selector transfer(...)

	cases := []struct {
		name string
		tx   Tx
		want TxKind
	}{
		{"creación de contrato", Tx{From: me, To: zero}, KindNew},
		{"llamada a contrato", Tx{From: me, To: other, Input: transferInput}, KindCall},
		{"self nativa", Tx{From: me, To: me, Value: big.NewInt(1)}, KindSelf},
		{"saliente nativa", Tx{From: me, To: other, Value: big.NewInt(1)}, KindOut},
		{"entrante nativa", Tx{From: other, To: me, Value: big.NewInt(1)}, KindIn},
		{"ajena", Tx{From: other, To: other, Value: big.NewInt(1)}, KindUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ClassifyTx(c.tx, me); got != c.want {
				t.Errorf("ClassifyTx = %v (%s), quiero %v (%s)", got, got, c.want, c.want)
			}
		})
	}
}
