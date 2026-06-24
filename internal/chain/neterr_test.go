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
		{"deadline", context.DeadlineExceeded, "request timed out"},
		{"deadline envuelto", fmt.Errorf("rpc: %w", context.DeadlineExceeded), "request timed out"},
		{"cancelado", context.Canceled, "canceled"},
		{"rate limit", errors.New("429 Too Many Requests"), "too many requests (rate-limited)"},
		{"dns tipado", &net.DNSError{Err: "no such host", Name: "x"}, "could not resolve host"},
		{"dns texto", errors.New("dial tcp: lookup foo: no such host"), "could not resolve host"},
		{"conn refused", errors.New("dial tcp 127.0.0.1:1: connect: connection refused"), "connection refused"},
		{"timeout texto", errors.New("net/http: request canceled (Client.Timeout exceeded)"), "request timed out"},
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
	if DescribeNetErr(ne) != "request timed out" {
		t.Errorf("net.Error con Timeout()=true no se clasificó como timeout")
	}
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o op" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }
