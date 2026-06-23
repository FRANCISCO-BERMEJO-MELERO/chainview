package ui

import (
	"testing"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/storage"
)

// TestNextLoadCancelsPrevious verifica que cada nueva carga cancela la anterior y
// avanza la generación (3.6).
func TestNextLoadCancelsPrevious(t *testing.T) {
	m := testModel(80, 24)

	ctx1, g1 := m.nextLoad()
	ctx2, g2 := m.nextLoad()

	if g2 <= g1 {
		t.Errorf("la generación debería crecer: g1=%d g2=%d", g1, g2)
	}
	if ctx1.Err() == nil {
		t.Error("nextLoad debería cancelar el contexto de la carga anterior")
	}
	if ctx2.Err() != nil {
		t.Error("el contexto de la carga nueva no debería estar cancelado")
	}

	// cancelLoad cancela el trabajo en vuelo sin iniciar otra carga.
	ctx3, _ := m.nextLoad()
	m.cancelLoad()
	if ctx3.Err() == nil {
		t.Error("cancelLoad debería cancelar el contexto en vuelo")
	}
}

// TestStaleResultsDiscarded comprueba que los mensajes de una generación vieja se
// ignoran y los de la actual se aplican (3.6).
func TestStaleResultsDiscarded(t *testing.T) {
	m := testModel(80, 24)
	m.prices = map[chain.PriceQuery]float64{}
	m.loadGen = 5

	q := chain.PriceQuery{ChainID: chain.ChainEthereum}

	// Generación vieja: se ignora.
	r, _ := m.Update(pricesMsg{gen: 4, prices: map[chain.PriceQuery]float64{q: 100}})
	if len(r.(Model).prices) != 0 {
		t.Error("un pricesMsg con generación vieja debería descartarse")
	}

	// Generación actual: se aplica.
	r, _ = m.Update(pricesMsg{gen: 5, prices: map[chain.PriceQuery]float64{q: 100}})
	if got := r.(Model).prices[q]; got != 100 {
		t.Errorf("un pricesMsg con la generación actual debería aplicarse, precio=%v", got)
	}

	// Un balancesMsg viejo tampoco muta los balances.
	m.balState = stateLoaded
	r, _ = m.Update(balancesMsg{gen: 4, results: []chain.BalanceResult{{ChainID: 1}}})
	if len(r.(Model).balResults) != 0 {
		t.Error("un balancesMsg con generación vieja debería descartarse")
	}
}

// TestLoadGenAdvancesOnWalletChange comprueba que mover el cursor de wallet avanza
// la generación, invalidando las cargas en vuelo de la wallet anterior (3.6).
func TestLoadGenAdvancesOnWalletChange(t *testing.T) {
	ws := &storage.Wallets{}
	_ = ws.Add("0x1111111111111111111111111111111111111111")
	_ = ws.Add("0x2222222222222222222222222222222222222222")
	m := testModel(80, 24)
	m.wallets = ws
	m.active = tabAccounts
	m.accCursor = 0

	start := m.loadGen
	r, _ := m.updateAccounts(keyMsg("down"))
	if r.(Model).loadGen <= start {
		t.Errorf("mover el cursor debería avanzar loadGen (era %d)", start)
	}
}
