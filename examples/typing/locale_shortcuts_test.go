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
	for _, form := range []osk.Form{osk.Full, osk.Compact} {
		if form == osk.Compact {
			d.Tap(ebiten.KeyF3)
		}
		for step := 1; step <= len(locales); step++ {
			locale := locales[(start+step)%len(locales)]
			d.Hold(4, ebiten.KeyF1)
			d.WithGame(func(ebitenGame ebiten.Game) {
				g := ebitenGame.(*game)
				if g.active != 1 || string(g.values[1]) != "name" {
					t.Errorf("locale switch left field %d active and email text %q", g.active, g.values[1])
				}
				checkVisibleStockLayout(t, g, locale, form, osk.KindEmail, osk.ActionNext)
			})
		}
	}
}

func TestTheExampleSwitchesLocaleAndFormOnTheNumberPad(t *testing.T) {
	d := ebitenmcp.T(t)
	d.Click(width*5/6, boardTop/2)
	d.Click(width/6, boardTop+37) // 1 in Number
	d.Tap(ebiten.KeyF1)
	d.WithGame(func(ebitenGame ebiten.Game) {
		checkVisibleStockLayout(t, ebitenGame.(*game), osk.EsES, osk.Compact, osk.KindNumber, osk.ActionDone)
	})
	d.Click(width/2, boardTop+37) // 2 in Number
	d.Tap(ebiten.KeyF3)
	d.Click(width*5/6, boardTop+37) // 3 in Number
	d.WithGame(func(ebitenGame ebiten.Game) {
		g := ebitenGame.(*game)
		if g.active != 2 || string(g.values[2]) != "123" {
			t.Errorf("layout switches left field %d active and number text %q", g.active, g.values[2])
		}
	})

	d.Click(width/6, boardTop/2) // Name uses the selected form.
	d.WithGame(func(ebitenGame ebiten.Game) {
		checkVisibleStockLayout(t, ebitenGame.(*game), osk.EsES, osk.Full, osk.KindText, osk.ActionNext)
	})
}

func checkVisibleStockLayout(t *testing.T, g *game, locale osk.Locale, form osk.Form, kind osk.Kind, action osk.FieldAction) {
	t.Helper()
	expected, err := osk.StockWithKind(locale, form, kind, action)
	if err != nil {
		t.Error(err)
		return
	}

	var want, got []string
	for _, row := range expected.Layers[0].Rows {
		for _, key := range row {
			if face := key.Face(); face != "" {
				want = append(want, face)
			}
		}
	}
	for _, key := range g.keyboard.AppendKeys(nil) {
		got = append(got, key.Face())
	}
	if !slices.Equal(got, want) {
		t.Errorf("visible %s %s keys = %q, want %q", locale, form, got, want)
	}
}
