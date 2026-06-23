package ui

import (
	"math/big"
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ethereum/go-ethereum/common"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/storage"
)

// --- 2.2 command palette ---

func TestFuzzyScore(t *testing.T) {
	if _, ok := fuzzyScore("", "lo que sea"); !ok {
		t.Error("query vacía debería casar siempre")
	}
	if _, ok := fuzzyScore("blc", "Balances"); !ok {
		t.Error("\"blc\" es subsecuencia de \"Balances\"")
	}
	if _, ok := fuzzyScore("xyz", "Balances"); ok {
		t.Error("\"xyz\" no debería casar con \"Balances\"")
	}
	// Coincidencia al inicio de palabra puntúa más que dispersa.
	start, _ := fuzzyScore("arb", "Arbitrum One")
	spread, _ := fuzzyScore("arb", "Para abrir")
	if start <= spread {
		t.Errorf("inicio de palabra (%d) debería puntuar más que disperso (%d)", start, spread)
	}
}

func TestFilteredCommandsRanksByQuery(t *testing.T) {
	m := testModel(80, 24)
	m.paletteInput = textinput.New()
	m.paletteInput.SetValue("balances")

	got := m.filteredCommands()
	if len(got) == 0 {
		t.Fatal("esperaba al menos un comando para \"balances\"")
	}
	if !strings.Contains(got[0].label, "Balances") {
		t.Errorf("primer resultado para \"balances\" = %q, esperaba uno con Balances", got[0].label)
	}
}

// --- 2.3 ordenación ---

func TestSortBalancesByValueDesc(t *testing.T) {
	m := balModelWithTokens() // de balances_test.go: 1 ETH = $2000 en Ethereum
	// Añadimos una segunda celda de menor valor en otra red.
	low := common.HexToAddress("0x2222222222222222222222222222222222222222")
	m.balResults = append(m.balResults, chain.BalanceResult{
		ChainID: chain.ChainBase, Address: low, Wei: big.NewInt(1_000_000_000_000_000), // 0.001 ETH
	})
	m.prices[chain.PriceQuery{ChainID: chain.ChainBase}] = 2000
	m.balSortKey = 1 // valor
	m.balSortAsc = false

	vis := append([]chain.BalanceResult(nil), m.balResults...)
	m.sortBalances(vis)
	if vis[0].ChainID != chain.ChainEthereum {
		t.Errorf("orden por valor desc: esperaba Ethereum (mayor) primero, got chain %d", vis[0].ChainID)
	}
}

func TestSortName(t *testing.T) {
	if balSortName(1) != "valor" || balSortName(0) != "" {
		t.Error("balSortName mal mapeado")
	}
	if txSortName(2) != "valor" || txSortName(0) != "fecha" {
		t.Error("txSortName mal mapeado")
	}
}

// --- 2.4 / 2.5 copiar y abrir ---

func TestBalCopyTarget(t *testing.T) {
	wallet := common.HexToAddress("0x1111111111111111111111111111111111111111")
	token := common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	if v, _ := balCopyTarget(balRow{address: wallet}); v != wallet.Hex() {
		t.Errorf("fila nativa debería copiar la address de la wallet, got %q", v)
	}
	tb := chain.TokenBalance{Token: token}
	if v, _ := balCopyTarget(balRow{address: wallet, token: &tb}); v != token.Hex() {
		t.Errorf("fila de token debería copiar la address del token, got %q", v)
	}
}

func TestExplorerURLs(t *testing.T) {
	m := testModel(80, 24) // networks = DefaultNetworks(), con Explorer poblado
	addr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	got, ok := m.explorerAddressURL(chain.ChainEthereum, addr)
	if !ok || !strings.HasPrefix(got, "https://etherscan.io/address/0x") {
		t.Errorf("URL de address mal construida: %q (ok=%v)", got, ok)
	}
	txURL, ok := m.explorerTxURL(chain.ChainEthereum, "0xabc")
	if !ok || txURL != "https://etherscan.io/tx/0xabc" {
		t.Errorf("URL de tx mal construida: %q", txURL)
	}
	// Red sin explorador → false.
	m.networks = []chain.Network{{ChainID: 999}}
	m.allNetworks = m.networks
	if _, ok := m.explorerAddressURL(999, addr); ok {
		t.Error("una red sin Explorer no debería dar URL")
	}
}

// --- 2.8 confirmación de borrado ---

func keyMsg(s string) tea.KeyPressMsg {
	switch s {
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Mod: tea.ModShift, Code: tea.KeyTab}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	}
	// ctrl+<letra>: p.ej. "ctrl+d", "ctrl+k", "ctrl+g".
	if rest, ok := strings.CutPrefix(s, "ctrl+"); ok && len(rest) == 1 {
		return tea.KeyPressMsg{Mod: tea.ModCtrl, Code: rune(rest[0])}
	}
	// Tecla imprimible simple (una runa): 'q', 'n', 's', '?'…
	return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
}

func TestConfirmDeleteRequiresTwoPresses(t *testing.T) {
	if keyMsg("ctrl+d").String() != "ctrl+d" {
		t.Fatalf("el helper de teclas no produce ctrl+d, got %q", keyMsg("ctrl+d").String())
	}
	ws := &storage.Wallets{}
	_ = ws.Add("0x1111111111111111111111111111111111111111")
	m := testModel(80, 24)
	m.wallets = ws
	m.active = tabAccounts

	// Primer ctrl+d: arma la confirmación, no borra.
	r1, _ := m.updateAccounts(keyMsg("ctrl+d"))
	m1 := r1.(Model)
	if !m1.confirmDel || m1.wallets.Len() != 1 {
		t.Fatalf("primer ctrl+d: confirmDel=%v len=%d, esperaba true/1", m1.confirmDel, m1.wallets.Len())
	}
	// esc cancela.
	r2, _ := m1.updateAccounts(keyMsg("esc"))
	m2 := r2.(Model)
	if m2.confirmDel || m2.wallets.Len() != 1 {
		t.Fatalf("esc debería cancelar la confirmación sin borrar")
	}
	// Dos ctrl+d seguidos: borra.
	r3, _ := m2.updateAccounts(keyMsg("ctrl+d"))
	r4, _ := r3.(Model).updateAccounts(keyMsg("ctrl+d"))
	if got := r4.(Model).wallets.Len(); got != 0 {
		t.Errorf("dos ctrl+d deberían borrar la wallet, quedan %d", got)
	}
}

// --- 2.9 indicador rate-limited ---

func TestRateLimitedSuffixNilClient(t *testing.T) {
	m := testModel(80, 24) // sin client
	if s := m.rateLimitedSuffix(); s != "" {
		t.Errorf("sin client el sufijo debería ser vacío, got %q", s)
	}
}

// --- render de overlays nuevos ---

func TestNewOverlaysFitFrame(t *testing.T) {
	cases := map[string]func(*Model){
		"paleta": func(m *Model) {
			m.paletteOpen = true
			m.paletteInput = textinput.New()
		},
		"tour": func(m *Model) {
			m.tourActive = true
			m.tourStep = 1
		},
	}
	for name, setup := range cases {
		m := testModel(80, 24)
		setup(&m)
		out := m.renderFrame()
		if w := lipgloss.Width(out); w != 80 {
			t.Errorf("%s: ancho del frame = %d, quiero 80", name, w)
		}
		if h := lipgloss.Height(out); h != 24 {
			t.Errorf("%s: alto del frame = %d, quiero 24", name, h)
		}
	}
}

// --- 2.10 tour ---

func TestTourAdvancesAndFinishes(t *testing.T) {
	m := testModel(80, 24)
	m.prefs = &storage.Prefs{}
	m.tourActive = true
	m.tourStep = 0
	steps := len(tourSteps())
	for i := 0; i < steps-1; i++ {
		r, _ := m.updateTour(keyMsg("enter"))
		m = r.(Model)
	}
	if m.tourStep != steps-1 || !m.tourActive {
		t.Fatalf("tras %d avances esperaba estar en el último paso aún activo", steps-1)
	}
	// Un enter más en el último paso termina el tour.
	r, _ := m.updateTour(keyMsg("enter"))
	if r.(Model).tourActive {
		t.Error("el último enter debería terminar el tour")
	}
}
