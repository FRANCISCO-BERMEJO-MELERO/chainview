package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestLayoutColumnsFlexFillsWidth(t *testing.T) {
	cols := []column{
		{title: "A", min: 10},
		{title: "B", min: 5, flex: true},
		{title: "C", min: 8, align: alignRight},
	}
	total := 60
	widths := layoutColumns(cols, total)

	// La columna fija conserva su mínimo; la flex absorbe el sobrante.
	if widths[0] != 10 || widths[2] != 8 {
		t.Errorf("columnas fijas alteradas: %v", widths)
	}
	if widths[1] <= 5 {
		t.Errorf("la columna flex no creció: %v", widths)
	}
	// La tabla debe ocupar exactamente el ancho dado (marcador + columnas + gaps).
	if got := tableWidth(widths); got != total {
		t.Errorf("tableWidth = %d, quiero %d (%v)", got, total, widths)
	}
}

func TestLayoutColumnsShrinksWhenNarrow(t *testing.T) {
	cols := []column{
		{title: "A", min: 20},
		{title: "B", min: 20, flex: true},
		{title: "C", min: 20},
	}
	widths := layoutColumns(cols, 30)
	for i, w := range widths {
		if w < 1 {
			t.Errorf("columna %d quedó por debajo de 1: %v", i, widths)
		}
	}
	// No debe exceder el total disponible aunque no quepan los mínimos.
	if got := tableWidth(widths); got > 30 {
		t.Errorf("tableWidth = %d desbordó 30 (%v)", got, widths)
	}
	// La flex es la primera en recortarse: debe quedar más estrecha que las fijas.
	if widths[1] >= widths[0] {
		t.Errorf("la flex no se recortó antes que la fija: %v", widths)
	}
}

func TestFitRightAlignsAndTruncates(t *testing.T) {
	// Relleno por la izquierda hasta el ancho exacto.
	if got := fitRight("12", 5); got != "   12" {
		t.Errorf("fitRight(\"12\",5) = %q, quiero %q", got, "   12")
	}
	// Trunca con … cuando no cabe.
	if got := fitRight("123456", 4); got != "123…" {
		t.Errorf("fitRight(\"123456\",4) = %q, quiero %q", got, "123…")
	}
	if lipgloss.Width(fitRight("muy largo", 5)) != 5 {
		t.Error("fitRight no respetó el ancho al truncar")
	}
}

func TestFitRightNumbersShareRightEdge(t *testing.T) {
	// Distintas magnitudes deben terminar en la misma columna (decimales alineados).
	nums := []string{"1.2345", "0.05", "2.87654321"}
	w := 12
	for _, n := range nums {
		cell := fitRight(n, w)
		if lipgloss.Width(cell) != w {
			t.Fatalf("celda %q ancho %d, quiero %d", cell, lipgloss.Width(cell), w)
		}
		if strings.HasSuffix(cell, " ") {
			t.Errorf("número no anclado a la derecha: %q", cell)
		}
	}
}

func TestTableRowSelectedSpansFullWidth(t *testing.T) {
	m := testModel(80, 24)
	cols := balanceColumns()
	widths := layoutColumns(cols, m.contentW)
	cells := []tcell{txt("vitalik.eth"), txt("Ethereum"), txt(""), txt("1.2345"), txt("ETH"), txt("$3,210.55")}

	row := m.tableRow(cols, widths, cells, true)
	if got := lipgloss.Width(row); got != tableWidth(widths) {
		t.Errorf("fila seleccionada ancho = %d, quiero %d", got, tableWidth(widths))
	}
	// El realce no debe depender solo del color: el marcador › sigue presente.
	if !strings.Contains(row, "›") {
		t.Error("la fila seleccionada perdió el marcador ›")
	}
}
