package main

import (
	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (g *game) chooseRegionalLayout() error {
	for i, locale := range [...]osk.Locale{osk.EnGB, osk.DeDE, osk.ItIT} {
		if inpututil.IsKeyJustPressed(ebiten.KeyF7 + ebiten.Key(i)) {
			form := osk.Compact
			if ebiten.IsKeyPressed(ebiten.KeyShift) {
				form = osk.Full
			}
			return g.selectStock(locale, form)
		}
	}
	return nil
}
