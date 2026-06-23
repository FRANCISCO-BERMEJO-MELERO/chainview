package chain

import (
	"math/big"
	"testing"
)

func TestFormatUnits(t *testing.T) {
	// bigStr parsea un literal decimal grande a *big.Int (los uint64 no llegan
	// a 1e18 cómodamente como literal con signo).
	bigStr := func(s string) *big.Int {
		v, ok := new(big.Int).SetString(s, 10)
		if !ok {
			t.Fatalf("literal inválido: %s", s)
		}
		return v
	}

	cases := []struct {
		name     string
		value    *big.Int
		decimals int
		want     string
	}{
		{"un ether exacto", bigStr("1000000000000000000"), 18, "1"},
		{"un ether y medio", bigStr("1500000000000000000"), 18, "1.5"},
		{"cuatro decimales", bigStr("1234500000000000000"), 18, "1.2345"},
		{"un wei sin notacion cientifica", big.NewInt(1), 18, "0.000000000000000001"},
		{"cero", big.NewInt(0), 18, "0"},
		{"nil es cero", nil, 18, "0"},
		{"negativo", bigStr("-2500000000000000000"), 18, "-2.5"},
		{"usdc 6 decimales", big.NewInt(123456), 6, "0.123456"},
		{"usdc cien", big.NewInt(100000000), 6, "100"},
		{"sin decimales devuelve el entero", big.NewInt(42), 0, "42"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatUnits(tc.value, tc.decimals); got != tc.want {
				t.Fatalf("FormatUnits(%v, %d) = %q, quiero %q", tc.value, tc.decimals, got, tc.want)
			}
		})
	}
}

func TestFormatEther(t *testing.T) {
	wei, _ := new(big.Int).SetString("2500000000000000000", 10)
	if got := FormatEther(wei); got != "2.5" {
		t.Fatalf("FormatEther = %q, quiero %q", got, "2.5")
	}
}

func TestFormatFiat(t *testing.T) {
	cases := []struct {
		value    float64
		currency string
		want     string
	}{
		{8421.07, "usd", "$8,421.07"},
		{1234567.5, "usd", "$1,234,567.50"},
		{0, "usd", "$0.00"},
		{12.3, "eur", "€12.30"},
		{999.99, "gbp", "999.99 GBP"}, // moneda sin símbolo: código sufijo
		{-5.5, "usd", "$-5.50"},
	}
	for _, c := range cases {
		if got := FormatFiat(c.value, c.currency); got != c.want {
			t.Errorf("FormatFiat(%v,%q) = %q, quiero %q", c.value, c.currency, got, c.want)
		}
	}
}
