package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// driver conduce un Model in-process: sustituye a teatest (incompatible con
// bubbletea v2) para pruebas de integración. Envía teclas/mensajes a través de
// Model.Update en secuencia y deja inspeccionar estado y frame tras cada paso.
type driver struct {
	t       *testing.T
	m       Model
	lastCmd tea.Cmd // el tea.Cmd devuelto por el último Update, por si hay que ejecutarlo
}

// newDriver arranca un driver sobre un Model de tamaño fijo (80×24 por defecto).
func newDriver(t *testing.T, m Model) *driver {
	t.Helper()
	return &driver{t: t, m: m}
}

// key envía una pulsación (por su nombre, p.ej. "tab", "ctrl+k", "q") y avanza.
func (d *driver) key(s string) *driver {
	d.t.Helper()
	return d.send(keyMsg(s))
}

// send inyecta un mensaje arbitrario (p.ej. balancesMsg) por Update.
func (d *driver) send(msg tea.Msg) *driver {
	d.t.Helper()
	r, cmd := d.m.Update(msg)
	d.m = r.(Model)
	d.lastCmd = cmd
	return d
}

// frame devuelve el render actual sin ANSI, para asertar contenido.
func (d *driver) frame() string { return plain(d.m.renderFrame()) }

// TestDriverTabNavigation recorre las pestañas con tab/shift+tab.
func TestDriverTabNavigation(t *testing.T) {
	d := newDriver(t, testModel(80, 24)) // arranca en Balances
	if d.m.active != tabBalances {
		t.Fatalf("estado inicial = %v, quiero Balances", d.m.active)
	}
	d.key("tab")
	if d.m.active != tabTransactions {
		t.Errorf("tras tab = %v, quiero Transacciones", d.m.active)
	}
	d.key("shift+tab").key("shift+tab")
	if d.m.active != tabAccounts {
		t.Errorf("tras 2×shift+tab = %v, quiero Cuentas", d.m.active)
	}
}

// TestDriverPaletteOpenClose abre la paleta con ctrl+k y la cierra con esc.
func TestDriverPaletteOpenClose(t *testing.T) {
	d := newDriver(t, testModel(80, 24))
	d.key("ctrl+k")
	if !d.m.paletteOpen {
		t.Fatal("ctrl+k debería abrir la paleta")
	}
	d.key("esc")
	if d.m.paletteOpen {
		t.Error("esc debería cerrar la paleta")
	}
}

// TestDriverHelpAndDebugOverlays comprueba el ruteo de los overlays globales.
func TestDriverHelpAndDebugOverlays(t *testing.T) {
	d := newDriver(t, testModel(80, 24))
	d.key("?")
	if !d.m.helpOpen {
		t.Fatal("? debería abrir la ayuda")
	}
	if !strings.Contains(d.frame(), "ayuda") {
		t.Errorf("el frame de ayuda no contiene 'ayuda':\n%s", d.frame())
	}
	d.key("esc")
	if d.m.helpOpen {
		t.Error("esc debería cerrar la ayuda")
	}
	d.key("ctrl+g")
	if !d.m.debugOpen {
		t.Error("ctrl+g debería abrir el overlay de debug")
	}
}

// TestDriverConfirmDelete reproduce la confirmación de borrado doble (2.x): el
// primer ctrl+d arma la confirmación y el segundo borra.
func TestDriverConfirmDelete(t *testing.T) {
	m := testModel(80, 24)
	m.active = tabAccounts
	_ = m.wallets.Add("0x1111111111111111111111111111111111111111")
	d := newDriver(t, m)

	d.key("ctrl+d")
	if !d.m.confirmDel {
		t.Fatal("primer ctrl+d debería armar la confirmación")
	}
	if d.m.wallets.Len() != 1 {
		t.Fatalf("no debería borrar todavía, len=%d", d.m.wallets.Len())
	}
	d.key("ctrl+d")
	if d.m.wallets.Len() != 0 {
		t.Errorf("segundo ctrl+d debería borrar, len=%d", d.m.wallets.Len())
	}
}

// TestDriverWelcomeFlow entra a la app desde la portada de bienvenida con enter.
func TestDriverWelcomeFlow(t *testing.T) {
	m := testModel(80, 24)
	m.showWelcome = true
	d := newDriver(t, m)
	d.key("enter")
	if d.m.showWelcome {
		t.Error("enter en la portada debería entrar a la app")
	}
}
