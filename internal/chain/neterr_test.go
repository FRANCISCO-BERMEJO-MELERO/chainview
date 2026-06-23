package chain

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
)

func TestDescribeNetErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"deadline", context.DeadlineExceeded, "tiempo de espera agotado"},
		{"deadline envuelto", fmt.Errorf("rpc: %w", context.DeadlineExceeded), "tiempo de espera agotado"},
		{"cancelado", context.Canceled, "cancelado"},
		{"rate limit", errors.New("429 Too Many Requests"), "demasiadas peticiones (rate-limited)"},
		{"dns tipado", &net.DNSError{Err: "no such host", Name: "x"}, "no se pudo resolver el host"},
		{"dns texto", errors.New("dial tcp: lookup foo: no such host"), "no se pudo resolver el host"},
		{"conn refused", errors.New("dial tcp 127.0.0.1:1: connect: connection refused"), "conexión rechazada"},
		{"timeout texto", errors.New("net/http: request canceled (Client.Timeout exceeded)"), "tiempo de espera agotado"},
		{"desconocido cae al texto", errors.New("algo raro"), "algo raro"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DescribeNetErr(tt.err); got != tt.want {
				t.Errorf("DescribeNetErr(%v) = %q, quiero %q", tt.err, got, tt.want)
			}
		})
	}
}

func TestCooldownDuration(t *testing.T) {
	// Siempre en [rpcCooldown, rpcCooldown+jitter). Muestreamos varias veces.
	for i := 0; i < 1000; i++ {
		d := cooldownDuration()
		if d < rpcCooldown || d >= rpcCooldown+rpcCooldownJitter {
			t.Fatalf("cooldownDuration = %v, fuera de [%v, %v)", d, rpcCooldown, rpcCooldown+rpcCooldownJitter)
		}
	}
}

// isTimeout debe reconocer net.Error con Timeout()==true aunque el texto no lo diga.
func TestIsTimeoutNetError(t *testing.T) {
	var ne net.Error = &timeoutErr{}
	if DescribeNetErr(ne) != "tiempo de espera agotado" {
		t.Errorf("net.Error con Timeout()=true no se clasificó como timeout")
	}
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o op" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }
