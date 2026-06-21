package ui

import (
	"math/big"
	"testing"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/ethereum/go-ethereum/common"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/storage"
)

func TestContentDims(t *testing.T) {
	w, h := contentDims(80, 24)
	if w != 76 || h != 16 {
		t.Errorf("contentDims(80,24) = (%d,%d), quiero (76,16)", w, h)
	}
	// Nunca devuelve dimensiones < 1, aunque el terminal sea diminuto.
	if w, h := contentDims(0, 0); w < 1 || h < 1 {
		t.Errorf("contentDims(0,0) = (%d,%d), debería ser >=1", w, h)
	}
}

func TestBar(t *testing.T) {
	got := bar("L", "R", 10)
	if lipgloss.Width(got) != 10 {
		t.Errorf("bar ancho = %d, quiero 10 (%q)", lipgloss.Width(got), got)
	}
	// Si no cabe, no debe desbordar el ancho dado.
	got = bar("izquierda-larga", "derecha-larga", 8)
	if lipgloss.Width(got) > 8 {
		t.Errorf("bar desbordó: ancho %d > 8 (%q)", lipgloss.Width(got), got)
	}
}

func testModel(w, h int) Model {
	m := Model{
		styles:   DefaultStyles(),
		wallets:  &storage.Wallets{},
		networks: chain.DefaultNetworks(),
		gas:      map[uint64]*big.Int{},
		gasPrev:  map[uint64]*big.Int{},
		ensNames: map[common.Address]string{},
		input:    textinput.New(),
		active:   tabBalances,
		balState: stateLoaded,
	}
	m.width, m.height = w, h
	m.contentW, m.contentH = contentDims(w, h)
	return m
}

func TestFrameFillsTerminal(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}, {60, 18}} {
		m := testModel(size[0], size[1])
		out := m.renderFrame()
		if gotW := lipgloss.Width(out); gotW != size[0] {
			t.Errorf("frame %dx%d: ancho = %d, quiero %d", size[0], size[1], gotW, size[0])
		}
		if gotH := lipgloss.Height(out); gotH != size[1] {
			t.Errorf("frame %dx%d: alto = %d, quiero %d", size[0], size[1], gotH, size[1])
		}
	}
}

func TestFrameDegradesWhenTooSmall(t *testing.T) {
	m := testModel(40, 10)
	out := m.renderFrame()
	if lipgloss.Width(out) > 40 || lipgloss.Height(out) > 10 {
		t.Errorf("el degradado desbordó: %dx%d", lipgloss.Width(out), lipgloss.Height(out))
	}
}
