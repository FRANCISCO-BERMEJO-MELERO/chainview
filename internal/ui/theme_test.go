package ui

import "testing"

func TestPaletteByName(t *testing.T) {
	if paletteByName("light") != paletteLight {
		t.Error("\"light\" debería dar la paleta clara")
	}
	// dark, auto y desconocidos caen a oscuro.
	for _, n := range []string{"dark", "auto", "rosa", ""} {
		if paletteByName(n) != paletteDark {
			t.Errorf("%q debería caer a la paleta oscura", n)
		}
	}
}

func TestStylesForUsesPalette(t *testing.T) {
	// Dos paletas distintas producen un color de marca distinto.
	dark := StylesFor(paletteDark).Brand.GetForeground()
	light := StylesFor(paletteLight).Brand.GetForeground()
	if dark == light {
		t.Error("el color de marca no debería coincidir entre tema claro y oscuro")
	}
}

func TestCycleThemeTogglesAndLeavesAuto(t *testing.T) {
	m := testModel(80, 24)
	m.themePref = "auto"
	m.themeNow = themeDark
	m.cycleTheme()
	if m.themeNow != themeLight || m.themePref != themeLight {
		t.Errorf("tras ciclar desde oscuro espero light/light, got %q/%q", m.themeNow, m.themePref)
	}
	m.cycleTheme()
	if m.themeNow != themeDark {
		t.Errorf("segundo ciclo debería volver a dark, got %q", m.themeNow)
	}
}
