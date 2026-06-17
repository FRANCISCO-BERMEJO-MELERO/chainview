package chain

import (
	"math/big"
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
