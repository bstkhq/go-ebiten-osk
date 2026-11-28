package main

import (
	"slices"
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestTheExampleCyclesLocalesAndWrapsOncePerPress(t *testing.T) {
	d := ebitenmcp.T(t)
	d.Click(width/2, boardTop/2)
	d.Type("name")
	locales := osk.StockLocales()
	start := slices.Index(locales, osk.EnUS)
	for step := 1; step <= len(locales); step++ {
		locale := locales[(start+step)%len(locales)]
		d.Hold(4, ebiten.KeyF1)
		d.WithGame(func(ebitenGame ebiten.Game) {
			g := ebitenGame.(*game)
			if g.locale != locale {
				t.Errorf("selected %s, want %s", g.locale, locale)
			}
			if g.active != 1 || string(g.values[1]) != "name" {
				t.Errorf("locale switch left field %d active and email text %q", g.active, g.values[1])
			}
			for _, key := range g.keyboard.AppendKeys(nil) {
				if key.Face() == "@" {
					return
				}
			}
			t.Error("the locale switch lost the full email layout")
		})
	}
}

func TestTheExampleSwitchesLocaleAndFormOnTheNumberPad(t *testing.T) {
	d := ebitenmcp.T(t)
	d.Click(width*5/6, boardTop/2)
	d.Click(width/6, boardTop+37) // 1 in Number
	d.Tap(ebiten.KeyF1)
	d.Click(width/2, boardTop+37) // 2 in Number
	d.Tap(ebiten.KeyF3)
	d.Click(width*5/6, boardTop+37) // 3 in Number
	d.WithGame(func(ebitenGame ebiten.Game) {
		g := ebitenGame.(*game)
		if g.active != 2 || string(g.values[2]) != "123" {
			t.Errorf("layout switches left field %d active and number text %q", g.active, g.values[2])
		}
	})
}
