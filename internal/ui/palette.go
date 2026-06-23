package ui

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// command es una entrada de la paleta de comandos (2.2): una etiqueta buscable y
// la acción que ejecuta sobre el Model.
type command struct {
	label string
	hint  string
	run   func(Model) (Model, tea.Cmd)
}

// fuzzyScore puntúa cuánto encaja query como subsecuencia de target (ambos
// case-insensitive). Devuelve (0, false) si query no es subsecuencia de target.
// Premia las coincidencias al inicio de palabra y las contiguas, para que
// "blc"→"Balances" puntúe alto y los aciertos dispersos bajo.
func fuzzyScore(query, target string) (int, bool) {
	if query == "" {
		return 0, true
	}
	q := strings.ToLower(query)
	t := strings.ToLower(target)

	score, ti, prev := 0, 0, -2
	for qi := 0; qi < len(q); qi++ {
		found := -1
		for ; ti < len(t); ti++ {
			if t[ti] == q[qi] {
				found = ti
				break
			}
		}
		if found == -1 {
			return 0, false
		}
		if found == 0 || t[found-1] == ' ' || t[found-1] == ':' {
			score += 3 // inicio de palabra
		}
		if found == prev+1 {
			score += 2 // contiguo al anterior
		}
		score++
		prev = found
		ti = found + 1
	}
	return score, true
}

// paletteCommands genera la lista de comandos según el contexto: navegación entre
// pestañas, salto a una wallet, conmutar redes y acciones (tema, recargar).
func (m Model) paletteCommands() []command {
	var cmds []command

	// Navegación entre pestañas.
	for _, t := range orderedTabs {
		t := t
		cmds = append(cmds, command{
			label: "Go to " + t.title(),
			hint:  "tab",
			run: func(m Model) (Model, tea.Cmd) {
				m.active = t
				return m, m.onEnterTab()
			},
		})
	}

	// Salto a una wallet: abre su detalle.
	for i, a := range m.wallets.List() {
		i, a := i, a
		cmds = append(cmds, command{
			label: "Wallet: " + m.displayName(a),
			hint:  "view detail",
			run: func(m Model) (Model, tea.Cmd) {
				m.accCursor = i
				m.active = tabAccounts
				cmd := m.onEnterTab()
				m.txViewport.SetContent(m.walletDetailContent(a))
				m.txViewport.GotoTop()
				m.walletDetailOpen = true
				return m, cmd
			},
		})
	}

	// Conmutar redes (activar/desactivar) desde el catálogo completo.
	active := map[uint64]bool{}
	for _, n := range m.networks {
		active[n.ChainID] = true
	}
	for _, n := range m.allNetworks {
		n := n
		state := "enable"
		if active[n.ChainID] {
			state = "disable"
		}
		cmds = append(cmds, command{
			label: "Network: " + n.Name + " (" + state + ")",
			hint:  "networks",
			run: func(m Model) (Model, tea.Cmd) {
				if !m.toggleNetwork(n.ChainID) {
					m.setNotice(noticeError, "At least one network must stay active")
					return m, noticeClearCmd(m.noticeUntil)
				}
				return m, m.onNetworksChanged()
			},
		})
	}

	// Acciones.
	cmds = append(cmds,
		command{
			label: "Theme: toggle light/dark",
			hint:  "action",
			run: func(m Model) (Model, tea.Cmd) {
				m.cycleTheme()
				return m, nil
			},
		},
		command{
			label: "Reload balances",
			hint:  "action",
			run: func(m Model) (Model, tea.Cmd) {
				if m.wallets.Len() == 0 {
					return m, nil
				}
				m.active = tabBalances
				m.balState = stateLoading // evita que onEnterTab dispare otra carga
				cmd := m.onEnterTab()
				ctx, gen := m.nextLoad()
				return m, tea.Batch(cmd, m.spinner.Tick, m.fetchBalancesCmd(ctx, gen))
			},
		},
	)
	return cmds
}

// filteredCommands devuelve los comandos que casan con el texto de la paleta,
// ordenados por puntuación difusa (mayor primero) y, a igualdad, por orden de
// generación (estable).
func (m Model) filteredCommands() []command {
	query := strings.TrimSpace(m.paletteInput.Value())
	type scored struct {
		cmd   command
		score int
		order int
	}
	var matches []scored
	for i, c := range m.paletteCommands() {
		if s, ok := fuzzyScore(query, c.label); ok {
			matches = append(matches, scored{c, s, i})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].order < matches[j].order
	})
	out := make([]command, len(matches))
	for i, s := range matches {
		out[i] = s.cmd
	}
	return out
}

// updatePalette maneja el teclado mientras la paleta está abierta.
func (m Model) updatePalette(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+k", "esc":
		m.paletteOpen = false
		return m, nil
	case "up":
		if m.paletteCursor > 0 {
			m.paletteCursor--
		}
		return m, nil
	case "down":
		if m.paletteCursor < len(m.filteredCommands())-1 {
			m.paletteCursor++
		}
		return m, nil
	case "enter":
		cmds := m.filteredCommands()
		m.paletteOpen = false
		if m.paletteCursor >= 0 && m.paletteCursor < len(cmds) {
			return cmds[m.paletteCursor].run(m)
		}
		return m, nil
	}
	// Cualquier otra tecla edita el filtro y reancla el cursor arriba.
	var cmd tea.Cmd
	m.paletteInput, cmd = m.paletteInput.Update(msg)
	m.paletteCursor = 0
	return m, cmd
}

// renderPalette dibuja el overlay de la paleta: campo de búsqueda + lista de
// comandos casados, centrado en el área de contenido.
func (m Model) renderPalette() string {
	var b strings.Builder
	b.WriteString(m.styles.StateTitle.Render("⌘  Command palette"))
	b.WriteString("\n\n")
	b.WriteString(m.paletteInput.View())
	b.WriteString("\n")

	cmds := m.filteredCommands()
	if len(cmds) == 0 {
		b.WriteString("\n" + m.styles.Faint.Render("No matches"))
	}

	// Mostramos hasta 'maxVisible' comandos alrededor del cursor.
	const maxVisible = 10
	start := 0
	if m.paletteCursor >= maxVisible {
		start = m.paletteCursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(cmds) {
		end = len(cmds)
	}
	for i := start; i < end; i++ {
		c := cmds[i]
		line := fit(c.label, 40) + m.styles.Faint.Render(c.hint)
		if i == m.paletteCursor {
			b.WriteString("\n" + m.styles.Balance.Render("› "+line))
		} else {
			b.WriteString("\n  " + line)
		}
	}
	if len(cmds) > end {
		b.WriteString("\n" + m.styles.Faint.Render(fmt.Sprintf("  … %d more", len(cmds)-end)))
	}

	panel := m.styles.Panel.Render(strings.TrimRight(b.String(), "\n"))
	return lipgloss.Place(m.contentW, m.contentH, lipgloss.Center, lipgloss.Center, panel)
}
