package main

import (
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestTheExampleSelectsBritishGermanAndItalianBoards(t *testing.T) {
	d := ebitenmcp.T(t)
	for _, test := range []struct {
		key    ebiten.Key
		locale osk.Locale
	}{
		{ebiten.KeyF7, "en-GB"},
		{ebiten.KeyF8, "de-DE"},
		{ebiten.KeyF9, "it-IT"},
	} {
		d.Tap(test.key)
		assertSelectedBoard(t, d, test.locale, osk.Compact, "Q")
		d.Tap(ebiten.KeyShiftLeft, test.key)
		assertSelectedBoard(t, d, test.locale, osk.Full, "q")
	}
}

func assertSelectedBoard(t *testing.T, d *ebitenmcp.Driver, locale osk.Locale, form osk.Form, firstKey string) {
	t.Helper()
	d.WithGame(func(ebitenGame ebiten.Game) {
		g := ebitenGame.(*game)
		if g.locale != locale || g.preferredForm != form {
			t.Errorf("selected %s %s, want %s %s", g.locale, g.preferredForm, locale, form)
		}
		if got := g.keyboard.AppendKeys(nil)[0].Face(); got != firstKey {
			t.Errorf("the %s %s board starts with %q, want %q", locale, form, got, firstKey)
		}
	})
}
