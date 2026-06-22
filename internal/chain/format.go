package chain

import (
	"math/big"
	"strconv"
	"strings"
)

// FormatUnits convierte un valor entero en su unidad base (p.ej. wei) a un
// string decimal legible, dividiendo por 10^decimals. Trabaja solo con big.Int
// y aritmética entera: nunca convierte a float, así que no pierde precisión ni
// cae en notación científica con valores muy pequeños o muy grandes.
//
// Ejemplos (decimals=18): 1e18 -> "1", 1.5e18 -> "1.5", 1 -> "0.000...001".
// Soporta cualquier nº de decimales para reutilizarlo con tokens ERC-20
// (USDC tiene 6, no 18).
func FormatUnits(value *big.Int, decimals int) string {
	if value == nil {
		return "0"
	}
	if decimals <= 0 {
		return value.String()
	}

	// Trabajamos con el valor absoluto y reponemos el signo al final.
	neg := value.Sign() < 0
	abs := new(big.Int).Abs(value)

	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	intPart, fracPart := new(big.Int).QuoRem(abs, divisor, new(big.Int))

	out := intPart.String()
	if fracPart.Sign() != 0 {
		// Rellenamos la parte fraccionaria con ceros a la izquierda hasta
		// `decimals` dígitos y quitamos los ceros sobrantes a la derecha.
		frac := fracPart.String()
		frac = strings.Repeat("0", decimals-len(frac)) + frac
		frac = strings.TrimRight(frac, "0")
		out += "." + frac
	}

	if neg {
		out = "-" + out
	}
	return out
}

// FormatEther es un atajo para FormatUnits con los 18 decimales del ETH.
func FormatEther(wei *big.Int) string {
	return FormatUnits(wei, 18)
}

// fiatSymbols asocia los códigos de moneda con su símbolo. Lo no listado se
// formatea con el código en mayúsculas como sufijo (p.ej. "1,234.56 GBP").
var fiatSymbols = map[string]string{
	"usd": "$",
	"eur": "€",
}

// FormatFiat formatea un importe en la moneda dada, con dos decimales y separador
// de miles, para que los valores de la cartera sean fáciles de leer y comparar.
// Ejemplo (usd): 8421.07 -> "$8,421.07".
func FormatFiat(value float64, currency string) string {
	currency = strings.ToLower(currency)
	num := groupThousands(strconv.FormatFloat(value, 'f', 2, 64))
	if sym, ok := fiatSymbols[currency]; ok {
		return sym + num
	}
	if currency == "" {
		return num
	}
	return num + " " + strings.ToUpper(currency)
}

// groupThousands inserta comas como separador de millares en la parte entera de
// un decimal ya formateado ("8421.07" -> "8,421.07"). Respeta el signo.
func groupThousands(s string) string {
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")

	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i:]
	}

	var b strings.Builder
	n := len(intPart)
	for i, ch := range intPart {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(ch)
	}

	out := b.String() + frac
	if neg {
		out = "-" + out
	}
	return out
}
