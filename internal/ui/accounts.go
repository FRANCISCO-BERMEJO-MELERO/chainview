package ui

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/ethereum/go-ethereum/common"

	"github.com/FRANCISCO-BERMEJO-MELERO/chainview/internal/chain"
)

// updateAccounts maneja las teclas de la pestaña Cuentas: escribir/añadir una
// address, navegar la lista y borrar la seleccionada. Las teclas de navegación
// (flechas, ctrl+d) no producen texto, así que conviven con el textinput sin
// modo aparte.
func (m Model) updateAccounts(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Con el detalle de wallet abierto: esc cierra, y/o copian/abren la address, el
	// resto de teclas las consume el viewport para desplazar.
	if m.walletDetailOpen {
		switch msg.String() {
		case "esc", "enter", "q":
			m.walletDetailOpen = false
			return m, nil
		case "y":
			if addr, ok := m.selectedWallet(); ok {
				return m, copyToClipboardCmd(addr.Hex(), "address")
			}
		case "o":
			if addr, ok := m.selectedWallet(); ok {
				if url, ok := m.explorerAddressURL(chain.ChainEthereum, addr); ok {
					return m, openURLCmd(url)
				}
			}
		}
		var cmd tea.Cmd
		m.txViewport, cmd = m.txViewport.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "enter":
		val := strings.TrimSpace(m.input.Value())
		// Input vacío + wallet seleccionada: enter abre el detalle de la wallet (2.7).
		if val == "" {
			if addr, ok := m.selectedWallet(); ok {
				m.txViewport.SetContent(m.walletDetailContent(addr))
				m.txViewport.GotoTop()
				m.walletDetailOpen = true
			}
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
		m.confirmDel = false
		m.input.Reset()
		m.balState = stateIdle // forzar recarga de balances con la nueva wallet
		m.clampAccCursor()
		return m, m.resolveWalletNamesCmd() // resolver el ENS de la nueva wallet

	case "ctrl+d", "delete":
		addrs := m.wallets.List()
		if m.accCursor < 0 || m.accCursor >= len(addrs) {
			return m, nil
		}
		addr := addrs[m.accCursor]
		// Segunda pulsación sobre la misma wallet: confirma y borra. Primera: arma
		// la confirmación (2.8), sin borrar todavía.
		if m.confirmDel && m.confirmDelAddr == addr {
			_ = m.wallets.Remove(addr)
			m.confirmDel = false
			m.balState = stateIdle
			m.clampAccCursor()
			return m, nil
		}
		m.confirmDel = true
		m.confirmDelAddr = addr
		return m, nil

	case "up":
		m.confirmDel = false // mover el cursor cancela una confirmación pendiente
		if m.accCursor > 0 {
			m.accCursor--
		}
		return m, nil

	case "down":
		m.confirmDel = false
		if m.accCursor < m.wallets.Len()-1 {
			m.accCursor++
		}
		return m, nil

	case "esc":
		if m.confirmDel {
			m.confirmDel = false // esc cancela la confirmación antes que limpiar el input
			return m, nil
		}
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

// walletDetailContent compone el agregado de una wallet entre redes (2.7): total
// fiat y, por red, su saldo nativo y tokens con su valor. Reutiliza los precios y
// balances ya cargados (sin red nueva).
func (m Model) walletDetailContent(addr common.Address) string {
	var b strings.Builder
	b.WriteString(m.styles.Balance.Render("Detalle de wallet") + "\n")
	b.WriteString(m.displayName(addr) + "\n")
	b.WriteString(m.styles.Faint.Render(addr.Hex()) + "\n\n")

	var results []chain.BalanceResult
	for _, r := range m.balResults {
		if r.Address == addr {
			results = append(results, r)
		}
	}
	if total, priced := m.visibleFiatTotal(results); priced {
		b.WriteString(m.styles.Faint.Render("Total  ") +
			m.styles.Balance.Render(chain.FormatFiat(total, m.fiatCurrency)) + "\n\n")
	}
	if len(results) == 0 {
		b.WriteString(m.styles.Faint.Render("Sin balances cargados aún — entra en la pestaña Balances."))
		return b.String()
	}

	for _, r := range results {
		b.WriteString(m.networkBadge(r.ChainID) + "  " + m.styles.Faint.Render(m.networkName(r.ChainID)) + "\n")
		nat := chain.FormatEther(r.Wei) + " " + m.networkSymbol(r.ChainID)
		b.WriteString("  " + fit(nat, 28))
		if v, ok := m.fiatValue(chain.PriceQuery{ChainID: r.ChainID}, r.Wei, 18); ok {
			b.WriteString(m.styles.Faint.Render("  " + chain.FormatFiat(v, m.fiatCurrency)))
		}
		b.WriteString("\n")
		for _, t := range r.Tokens {
			amt := chain.FormatUnits(t.Balance, int(t.Decimals)) + " " + t.Symbol
			b.WriteString("    ↳ " + fit(amt, 26))
			if v, ok := m.fiatValue(chain.PriceQuery{ChainID: r.ChainID, Token: t.Token}, t.Balance, t.Decimals); ok {
				b.WriteString(m.styles.Faint.Render("  " + chain.FormatFiat(v, m.fiatCurrency)))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString(m.styles.Faint.Render("y copiar address · o abrir en explorador · esc cerrar"))
	return b.String()
}

func (m Model) renderAccounts() string {
	if m.walletDetailOpen {
		return m.txViewport.View()
	}

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
	if m.confirmDel {
		b.WriteString(m.styles.NoticeError.Render("¿Borrar "+m.displayName(m.confirmDelAddr)+"?") +
			m.styles.Faint.Render("  ctrl+d confirma · esc cancela"))
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
