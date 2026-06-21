package ui

import (
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/storage"
)

func netModel() Model {
	m := testModel(80, 24)
	m.allNetworks = chain.DefaultNetworks()
	m.networks = chain.DefaultNetworks()
	m.prefs = &storage.Prefs{}
	return m
}

func TestEnabledNetworksDefaults(t *testing.T) {
	all := chain.DefaultNetworks()
	// Sin prefs: todas activas.
	if got := enabledNetworks(all, nil); len(got) != len(all) {
		t.Errorf("sin prefs = %d redes, quiero %d", len(got), len(all))
	}
	// Prefs vacías: todas activas (lista vacía = default).
	if got := enabledNetworks(all, &storage.Prefs{}); len(got) != len(all) {
		t.Errorf("prefs vacías = %d redes, quiero %d", len(got), len(all))
	}
	// Selección explícita: solo esas, en orden de catálogo.
	p := &storage.Prefs{EnabledChains: []uint64{chain.ChainBase, chain.ChainEthereum}}
	got := enabledNetworks(all, p)
	if len(got) != 2 || got[0].ChainID != chain.ChainEthereum || got[1].ChainID != chain.ChainBase {
		t.Errorf("selección explícita mal filtrada/ordenada: %+v", got)
	}
}

func TestToggleNetworkDisablesAndPersists(t *testing.T) {
	m := netModel()
	if ok := m.toggleNetwork(chain.ChainBase); !ok {
		t.Fatal("desactivar Base debería poder")
	}
	if len(m.networks) != 3 {
		t.Errorf("tras desactivar Base hay %d redes, quiero 3", len(m.networks))
	}
	for _, n := range m.networks {
		if n.ChainID == chain.ChainBase {
			t.Error("Base sigue activa tras desactivarla")
		}
	}
	if m.prefs.IsChainEnabled(chain.ChainBase) {
		t.Error("la preferencia no reflejó la desactivación de Base")
	}
	// Reactivar la vuelve a poner (en orden de catálogo).
	if ok := m.toggleNetwork(chain.ChainBase); !ok {
		t.Fatal("reactivar Base debería poder")
	}
	if len(m.networks) != 4 {
		t.Errorf("tras reactivar hay %d redes, quiero 4", len(m.networks))
	}
}

func TestToggleNetworkKeepsAtLeastOne(t *testing.T) {
	m := netModel()
	// Dejar solo Ethereum activa.
	for _, id := range []uint64{chain.ChainArbitrum, chain.ChainBase, chain.ChainOptimism} {
		m.toggleNetwork(id)
	}
	if len(m.networks) != 1 {
		t.Fatalf("preparación: quedan %d redes, quiero 1", len(m.networks))
	}
	// Desactivar la última debe fallar y no tocar nada.
	if ok := m.toggleNetwork(chain.ChainEthereum); ok {
		t.Error("desactivar la última red debería devolver false")
	}
	if len(m.networks) != 1 {
		t.Errorf("la última red se desactivó: quedan %d", len(m.networks))
	}
}

func TestNetworksOverlayFitsContent(t *testing.T) {
	m := netModel()
	m.networksOpen = true
	out := m.renderFrame()
	if w := lipgloss.Width(out); w != 80 {
		t.Errorf("frame con overlay ancho %d, quiero 80", w)
	}
	if h := lipgloss.Height(out); h != 24 {
		t.Errorf("frame con overlay alto %d, quiero 24", h)
	}
}
