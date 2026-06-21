package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
)

func TestStatusRightFitsWidth(t *testing.T) {
	m := testModel(80, 24)
	m.networks = chain.DefaultNetworks()
	m.active = tabTransactions
	// A cualquier ancho razonable, la leyenda no debe exceder el disponible.
	for _, avail := range []int{20, 40, 60, 100} {
		got := m.statusRight(avail)
		if w := lipgloss.Width(got); w > avail && avail >= lipgloss.Width("? ayuda · q salir") {
			t.Errorf("avail=%d: leyenda ancho %d desborda", avail, w)
		}
		// Siempre incluye el acceso a la ayuda.
		if !strings.Contains(got, "? ayuda") {
			t.Errorf("avail=%d: falta '? ayuda' en %q", avail, got)
		}
	}
}

func TestHintItemsAllHaveVerb(t *testing.T) {
	m := testModel(80, 24)
	m.networks = chain.DefaultNetworks()
	for _, tb := range []tab{tabAccounts, tabBalances, tabTransactions} {
		m.active = tb
		for _, it := range m.hintItems() {
			// Cada atajo es "tecla acción": debe llevar al menos un espacio, para que
			// nunca aparezca una tecla suelta sin explicar (p.ej. "r" a secas).
			if !strings.Contains(strings.TrimSpace(it), " ") {
				t.Errorf("tab %d: atajo sin acción: %q", tb, it)
			}
		}
	}
}

func TestStatusRightWideShowsMoreThanNarrow(t *testing.T) {
	m := testModel(80, 24)
	m.networks = chain.DefaultNetworks()
	m.active = tabTransactions
	narrow := m.statusRight(45)
	wide := m.statusRight(200)
	if lipgloss.Width(wide) <= lipgloss.Width(narrow) {
		t.Errorf("la leyenda ancha (%d) no muestra más que la estrecha (%d)",
			lipgloss.Width(wide), lipgloss.Width(narrow))
	}
}
