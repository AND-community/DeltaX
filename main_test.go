package main

import (
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/lucian95511/and/internal/pluginapi"
	"github.com/lucian95511/and/internal/ui"
)

func TestUygulaTemaRenkleriUygular(t *testing.T) {
	uygulaTema(pluginapi.ThemeReq{Accent: "205", SelBG: "162", SelFG: "255", Muted: "182"})

	if got := stTitle.GetForeground(); got != lipgloss.Color("205") {
		t.Errorf("stTitle ön plan rengi = %v, beklenen 205", got)
	}
	if got := stFocused.GetForeground(); got != lipgloss.Color("205") {
		t.Errorf("stFocused ön plan rengi = %v, beklenen 205", got)
	}
	if got := stSel.GetBackground(); got != lipgloss.Color("162") {
		t.Errorf("stSel arka plan rengi = %v, beklenen 162", got)
	}
	if got := stDim.GetForeground(); got != lipgloss.Color("182") {
		t.Errorf("stDim ön plan rengi = %v, beklenen 182", got)
	}
}

// TestUygulaTemaBosAlanlarVarsayilanaDuser: boş tema, DeltaX'e özgü değil
// AND'ın ortak varsayılan temasına (internal/ui) düşmeli — eklentiler artık
// kendi ayrı paletlerini taşımıyor.
func TestUygulaTemaBosAlanlarVarsayilanaDuser(t *testing.T) {
	uygulaTema(pluginapi.ThemeReq{})

	d := ui.DefaultTheme()
	if got := stTitle.GetForeground(); got != lipgloss.Color(d.Accent) {
		t.Errorf("boş temada varsayılan accent (%s) bekleniyordu, %v geldi", d.Accent, got)
	}
	if got := stSel.GetBackground(); got != lipgloss.Color(d.SelBG) {
		t.Errorf("boş temada varsayılan selBG (%s) bekleniyordu, %v geldi", d.SelBG, got)
	}
}
