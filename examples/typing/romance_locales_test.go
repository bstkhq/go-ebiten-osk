package main

import (
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestTheExampleSelectsFrenchPortugueseAndCatalan(t *testing.T) {
	d := ebitenmcp.T(t)
	for _, shortcut := range []ebiten.Key{ebiten.KeyF10, ebiten.KeyF11, ebiten.KeyF12} {
		d.Tap(shortcut)
		d.Click(48, boardTop+37)
		d.KeyDown(ebiten.KeyShiftLeft)
		d.Tap(shortcut)
		d.KeyUp(ebiten.KeyShiftLeft)
		d.Click(48, boardTop+37)
		if shortcut != ebiten.KeyF10 {
			d.Click(width-48, boardTop+112) // ç closes the second row.
		}
	}
	d.WithGame(func(g ebiten.Game) {
		if got := string(g.(*game).values[0]); got != "AaQqçQqç" {
			t.Errorf("switching locales and forms left %q, want AaQqçQqç", got)
		}
	})
}
