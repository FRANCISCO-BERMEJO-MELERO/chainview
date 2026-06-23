package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// tourStep es un paso del recorrido guiado de primera vez (2.10).
type tourStep struct {
	title string
	body  string
}

// tourSteps es el guion del tour: pocos pasos que señalan lo esencial.
func tourSteps() []tourStep {
	return []tourStep{
		{"Pestañas", "Tienes tres pestañas: Cuentas, Balances y Transacciones.\nCambia entre ellas con tab / shift+tab."},
		{"Paleta de comandos", "Pulsa ctrl+k en cualquier momento para buscar wallets,\nredes y acciones sin recordar atajos."},
		{"Redes y tema", "n elige qué redes ver. El tema se adapta al fondo del\nterminal; puedes alternarlo con ctrl+k → «Tema»."},
		{"Copiar y abrir", "En Balances y Transacciones, y copia el dato seleccionado\nal portapapeles y o lo abre en el explorador de bloques."},
	}
}

// maybeStartTour activa el tour si procede: hay prefs, no se ha hecho aún, y no
// estamos mostrando la portada (que se encarga de lanzarlo al cerrarse).
func (m *Model) maybeStartTour() {
	if m.prefs != nil && !m.prefs.TourDone {
		m.tourActive = true
		m.tourStep = 0
	}
}

// updateTour maneja el teclado durante el tour: avanzar, o saltar/terminar.
func (m Model) updateTour(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", " ", "right", "l":
		if m.tourStep < len(tourSteps())-1 {
			m.tourStep++
			return m, nil
		}
		m.finishTour() // último paso: terminar
	case "left", "h":
		if m.tourStep > 0 {
			m.tourStep--
		}
	case "esc", "q":
		m.finishTour() // saltar
	}
	return m, nil
}

// finishTour cierra el tour y lo marca como hecho para no repetirlo.
func (m *Model) finishTour() {
	m.tourActive = false
	if m.prefs != nil {
		_ = m.prefs.SetTourDone(true)
	}
}

// renderTour dibuja el paso actual del tour centrado en el área de contenido.
func (m Model) renderTour() string {
	steps := tourSteps()
	if m.tourStep < 0 || m.tourStep >= len(steps) {
		return ""
	}
	s := steps[m.tourStep]

	var b strings.Builder
	b.WriteString(m.styles.Brand.Render("✦ "+s.title) + "\n\n")
	b.WriteString(s.body + "\n\n")
	b.WriteString(m.styles.Faint.Render(fmt.Sprintf("Paso %d/%d", m.tourStep+1, len(steps))))
	b.WriteString(m.styles.Faint.Render("   ·   enter siguiente · esc saltar"))

	panel := m.styles.Panel.Render(b.String())
	return lipgloss.Place(m.contentW, m.contentH, lipgloss.Center, lipgloss.Center, panel)
}
