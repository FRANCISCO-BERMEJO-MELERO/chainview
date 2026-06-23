package chain

import (
	"context"
	"errors"
	"net"
	"strings"
)

// DescribeNetErr traduce un error de red al lenguaje del usuario, por tipo, en vez
// de mostrar el texto crudo del driver (que suele ser ruidoso e inglés técnico).
// Es best-effort: si no reconoce el error, devuelve su texto tal cual. Reconoce
// también errores envueltos (fmt.Errorf("...: %w", err)).
func DescribeNetErr(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "tiempo de espera agotado"
	case errors.Is(err, context.Canceled):
		return "cancelado"
	case isRateLimit(err):
		return "demasiadas peticiones (rate-limited)"
	case isDNSError(err):
		return "no se pudo resolver el host"
	case isConnRefused(err):
		return "conexión rechazada"
	case isTimeout(err):
		return "tiempo de espera agotado"
	default:
		return err.Error()
	}
}

// isDNSError detecta un fallo de resolución de nombre, ya sea por el tipo tipado
// de la stdlib o por el texto (algunos drivers lo envuelven perdiendo el tipo).
func isDNSError(err error) bool {
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "no such host")
}

// isConnRefused detecta una conexión rechazada (puerto cerrado / servicio caído).
func isConnRefused(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "connection refused")
}

// isTimeout detecta timeouts de red genéricos (net.Error.Timeout) además del
// context.DeadlineExceeded que ya cubre el caller.
func isTimeout(err error) bool {
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "timeout")
}
