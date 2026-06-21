package ui

import (
	"math/big"
	"testing"
)

func TestGwei(t *testing.T) {
	cases := []struct {
		wei  int64 // gas price en wei
		want string
	}{
		{136932722, "0.1369"}, // mainnet sub-gwei, truncado a 4 decimales
		{20244000, "0.0202"},  // L2 con cola larga, truncada
		{6000000, "0.006"},    // Base: sub-0.01, NO debe quedar en "0.00"
		{1000000, "0.001"},    // OP: aún más bajo
		{25000000000, "25"},   // mainnet congestionada, entero limpio
		{1500000000, "1.5"},   // 1.5 gwei
		{0, "0"},              // sin datos
	}
	for _, c := range cases {
		if got := gwei(big.NewInt(c.wei)); got != c.want {
			t.Errorf("gwei(%d) = %q, quiero %q", c.wei, got, c.want)
		}
	}
}
