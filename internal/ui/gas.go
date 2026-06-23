package ui

import (
	"context"
	"math/big"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
)

// gasTickMsg lo emite el tick periódico de refresco del gas. A diferencia del de
// balances, el gas se refresca siempre (el header está visible en todas las tabs).
type gasTickMsg struct{}

// gasMsg transporta el resultado del fetch concurrente de gas por red.
type gasMsg struct {
	results []chain.GasResult
}

// gasTickCmd programa el siguiente tick de refresco de gas.
func gasTickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return gasTickMsg{} })
}

// fetchGasCmd lanza el fetch concurrente del gas de todas las redes en background.
func (m Model) fetchGasCmd() tea.Cmd {
	client := m.client
	networks := m.networks
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return gasMsg{results: client.GasPrices(ctx, networks)}
	}
}

// gasLabels son las etiquetas cortas por red para que el header quepa en una línea.
var gasLabels = map[uint64]string{
	chain.ChainEthereum: "ETH",
	chain.ChainArbitrum: "ARB",
	chain.ChainBase:     "BASE",
	chain.ChainOptimism: "OP",
	chain.ChainPolygon:  "POL",
	chain.ChainScroll:   "SCR",
	chain.ChainLinea:    "LINEA",
}

func gasLabel(chainID uint64) string {
	if l, ok := gasLabels[chainID]; ok {
		return l
	}
	return "?"
}

// gwei formatea un gas price (wei) a gwei con hasta 4 decimales, quitando los
// ceros finales. Las L2 tienen gas sub-0.01 gwei (p.ej. 0.001), así que limitar a
// pocos decimales lo redondearía a "0.00" y perdería la información.
func gwei(wei *big.Int) string {
	s := chain.FormatUnits(wei, 9) // ya viene sin ceros finales
	if dot := strings.IndexByte(s, '.'); dot >= 0 && len(s)-(dot+1) > 4 {
		s = strings.TrimRight(s[:dot+5], "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

// renderGasHeader pinta el gas + tendencia de cada red en una sola línea.
func (m Model) renderGasHeader() string {
	if len(m.gas) == 0 {
		return m.styles.Faint.Render("⛽ gas: cargando…")
	}

	parts := make([]string, 0, len(m.networks))
	for _, n := range m.networks {
		wei, ok := m.gas[n.ChainID]
		if !ok {
			continue
		}

		cell := m.styles.Faint.Render(gasLabel(n.ChainID)+" ") + gwei(wei)
		// Tendencia respecto a la lectura anterior: ↑ rojo (sube), ↓ verde (baja),
		// = neutro. Solo aparece cuando ya hay una lectura previa con la que comparar.
		if prev, ok := m.gasPrev[n.ChainID]; ok {
			switch wei.Cmp(prev) {
			case 1:
				cell += m.styles.TrendUp.Render(" ↑")
			case -1:
				cell += m.styles.TrendDown.Render(" ↓")
			default:
				cell += m.styles.Faint.Render(" =")
			}
		}
		parts = append(parts, cell)
	}

	return m.styles.Faint.Render("⛽ ") + strings.Join(parts, m.styles.Faint.Render("   ")) + m.styles.Faint.Render(" gwei")
}
