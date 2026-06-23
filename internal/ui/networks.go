package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/storage"
)

// enabledNetworks filtra el catálogo por las redes que el usuario tiene activas
// en sus preferencias. Sin prefs (p.ej. en tests) o sin ninguna activa, devuelve
// el catálogo completo: nunca se queda sin redes.
func enabledNetworks(all []chain.Network, prefs *storage.Prefs) []chain.Network {
	if prefs == nil {
		return all
	}
	out := make([]chain.Network, 0, len(all))
	for _, n := range all {
		if prefs.IsChainEnabled(n.ChainID) {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return all
	}
	return out
}

// filterNetworks devuelve las redes del catálogo cuyo chain ID está en `ids`,
// respetando el orden de presentación del catálogo.
func filterNetworks(all []chain.Network, ids []uint64) []chain.Network {
	want := make(map[uint64]bool, len(ids))
	for _, id := range ids {
		want[id] = true
	}
	out := make([]chain.Network, 0, len(ids))
	for _, n := range all {
		if want[n.ChainID] {
			out = append(out, n)
		}
	}
	return out
}

// updateNetworks maneja el teclado del overlay de redes: navegar, conmutar y
// cerrar. Recibe puntero indirecto vía el valor de Model que devuelve.
func (m Model) updateNetworks(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "n", "esc", "q":
		m.networksOpen = false
		return m, nil
	case "up":
		if m.networksCursor > 0 {
			m.networksCursor--
		}
		return m, nil
	case "down":
		if m.networksCursor < len(m.allNetworks)-1 {
			m.networksCursor++
		}
		return m, nil
	case " ", "enter":
		if m.networksCursor < 0 || m.networksCursor >= len(m.allNetworks) {
			return m, nil
		}
		id := m.allNetworks[m.networksCursor].ChainID
		if !m.toggleNetwork(id) {
			m.setNotice(noticeError, "Debe quedar al menos una red activa")
			return m, noticeClearCmd(m.noticeUntil)
		}
		return m, m.onNetworksChanged()
	}
	return m, nil
}

// toggleNetwork activa/desactiva una red, persiste la selección y recalcula la
// vista filtrada. Devuelve false (sin tocar nada) si la acción dejaría cero
// redes activas.
func (m *Model) toggleNetwork(id uint64) bool {
	enabled := make(map[uint64]bool, len(m.networks))
	for _, n := range m.networks {
		enabled[n.ChainID] = true
	}

	if enabled[id] {
		if len(enabled) <= 1 {
			return false // no se puede desactivar la última red
		}
		delete(enabled, id)
	} else {
		enabled[id] = true
	}

	ids := make([]uint64, 0, len(enabled))
	for _, n := range m.allNetworks {
		if enabled[n.ChainID] {
			ids = append(ids, n.ChainID)
		}
	}
	if m.prefs != nil {
		_ = m.prefs.SetEnabledChains(ids)
	}
	m.networks = filterNetworks(m.allNetworks, ids)
	return true
}

// onNetworksChanged refresca lo que depende de las redes activas: el gas (header
// siempre visible), los balances y el historial de transacciones (multi-red).
// Recarga de inmediato la pestaña activa; las demás se recargan al entrar.
func (m *Model) onNetworksChanged() tea.Cmd {
	cmds := []tea.Cmd{m.fetchGasCmd()}

	m.balState = stateIdle
	if m.active == tabBalances && m.wallets.Len() > 0 {
		m.balState = stateLoading
		ctx, gen := m.nextLoad()
		cmds = append(cmds, m.spinner.Tick, m.fetchBalancesCmd(ctx, gen))
	}

	// El historial multi-red depende de las redes activas: lo invalidamos. Si
	// algún filtro de red apuntaba a una red ya desactivada, lo reseteamos.
	if m.txNetFilter != 0 && !m.prefsChainEnabled(m.txNetFilter) {
		m.txNetFilter = 0
	}
	m.txState = stateIdle
	if m.active == tabTransactions {
		if cmd := m.loadTxsCmd(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

// prefsChainEnabled indica si una red sigue entre las activas.
func (m Model) prefsChainEnabled(id uint64) bool {
	for _, n := range m.networks {
		if n.ChainID == id {
			return true
		}
	}
	return false
}

// renderNetworks dibuja el overlay de selección de redes centrado en el área de
// contenido, con el mismo estilo de panel que la ayuda.
func (m Model) renderNetworks() string {
	enabled := make(map[uint64]bool, len(m.networks))
	for _, n := range m.networks {
		enabled[n.ChainID] = true
	}

	var b strings.Builder
	b.WriteString(m.styles.StateTitle.Render("🌐  Redes"))
	b.WriteString("\n")

	for i, n := range m.allNetworks {
		mark := "[ ]"
		if enabled[n.ChainID] {
			mark = "[x]"
		}
		if i == m.networksCursor {
			// Fila seleccionada: barra a ancho completo (texto plano), igual que en
			// las tablas, para que el realce no dependa solo del color.
			b.WriteString(m.styles.RowSelected.Render("› " + mark + " " + n.Name))
		} else {
			styledMark := mark
			if enabled[n.ChainID] {
				styledMark = m.styles.Balance.Render(mark)
			}
			b.WriteString("  " + styledMark + " " + n.Name)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.styles.Faint.Render("espacio conmutar · esc cerrar"))

	panel := m.styles.Panel.Render(strings.TrimRight(b.String(), "\n"))
	return lipgloss.Place(m.contentW, m.contentH, lipgloss.Center, lipgloss.Center, panel)
}
