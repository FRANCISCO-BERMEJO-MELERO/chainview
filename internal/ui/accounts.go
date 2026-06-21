package ui

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/ethereum/go-ethereum/common"
)

// updateAccounts maneja las teclas de la pestaña Cuentas: escribir/añadir una
// address, navegar la lista y borrar la seleccionada. Las teclas de navegación
// (flechas, ctrl+d) no producen texto, así que conviven con el textinput sin
// modo aparte.
func (m Model) updateAccounts(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		val := strings.TrimSpace(m.input.Value())
		if val == "" {
			return m, nil
		}
		// Si parece un nombre ENS (no es una address hex y lleva punto, p.ej.
		// "vitalik.eth"), lo resolvemos primero y añadimos la address resultante.
		if !common.IsHexAddress(val) && strings.Contains(val, ".") {
			if m.ens == nil {
				m.addErr = errors.New("resolución ENS no disponible")
				return m, nil
			}
			m.addErr = nil
			m.resolvingName = val
			return m, m.resolveAndAddCmd(val)
		}
		// Address hex (o entrada inválida, que Add rechazará con su propio mensaje).
		if err := m.wallets.Add(val); err != nil {
			m.addErr = err
			return m, nil
		}
		m.addErr = nil
		m.input.Reset()
		m.balState = stateIdle // forzar recarga de balances con la nueva wallet
		m.clampAccCursor()
		return m, m.resolveWalletNamesCmd() // resolver el ENS de la nueva wallet

	case "ctrl+d", "delete":
		addrs := m.wallets.List()
		if m.accCursor >= 0 && m.accCursor < len(addrs) {
			_ = m.wallets.Remove(addrs[m.accCursor])
			m.balState = stateIdle
			m.clampAccCursor()
		}
		return m, nil

	case "up":
		if m.accCursor > 0 {
			m.accCursor--
		}
		return m, nil

	case "down":
		if m.accCursor < m.wallets.Len()-1 {
			m.accCursor++
		}
		return m, nil

	case "esc":
		m.input.Reset()
		m.addErr = nil
		return m, nil
	}

	// Cualquier otra tecla va al textinput (edición de la address).
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// clampAccCursor mantiene el cursor de la lista dentro de rango tras añadir o
// borrar wallets.
func (m *Model) clampAccCursor() {
	n := m.wallets.Len()
	if n == 0 {
		m.accCursor = 0
		return
	}
	if m.accCursor >= n {
		m.accCursor = n - 1
	}
	if m.accCursor < 0 {
		m.accCursor = 0
	}
}

func (m Model) renderAccounts() string {
	var b strings.Builder

	b.WriteString(m.input.View())
	b.WriteString("\n")
	if m.addErr != nil {
		b.WriteString(m.styles.Error.Render(m.addErr.Error()))
		b.WriteString("\n")
	}
	if m.resolvingName != "" {
		b.WriteString(m.styles.Faint.Render("resolviendo " + m.resolvingName + "…"))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	addrs := m.wallets.List()
	if len(addrs) == 0 {
		b.WriteString(m.styles.Faint.Render("Sin wallets — escribe una address (0x…) o un nombre ENS y pulsa Enter."))
		return b.String()
	}

	for i, a := range addrs {
		// Con ENS mostramos el nombre como principal y la address como secundaria;
		// sin ENS, la address completa (es el dato canónico de la wallet).
		label := a.Hex()
		if name := m.ensNames[a]; name != "" {
			label = name + " (" + shortAddr(a) + ")"
		}
		if i == m.accCursor {
			b.WriteString(m.styles.Balance.Render("› " + label))
		} else {
			b.WriteString("  " + label)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.styles.Faint.Render(fmt.Sprintf("%d wallet(s) seguida(s)", len(addrs))))
	return b.String()
}
