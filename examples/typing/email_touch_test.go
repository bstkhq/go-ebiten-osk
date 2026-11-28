package main

import (
	"image"
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestSpanishTextAndEmailFromTouch(t *testing.T) {
	d := ebitenmcp.T(t)
	inject := d.Runtime().Injector()
	touch := func(at image.Point) {
		inject.SetTouches([]ebitenmcp.Touch{{ID: 1, X: at.X, Y: at.Y}})
		d.Tick(2)
		inject.SetTouches(nil)
		d.Tick(2)
	}
	key := func(face string) {
		t.Helper()
		var at image.Point
		d.WithGame(func(g ebiten.Game) {
			for _, key := range g.(*game).keyboard.AppendKeys(nil) {
				if key.Face() == face {
					at = key.Rect.Min.Add(key.Rect.Size().Div(2))
					return
				}
			}
		})
		if at == (image.Point{}) {
			t.Fatalf("the keyboard shows no key %q", face)
		}
		touch(at)
	}

	d.Tap(ebiten.KeyF1) // es-ES is next after en-US.
	d.Tap(ebiten.KeyF3) // Full.
	base := keyCenter(t, d, "a", false)
	inject.SetTouches([]ebitenmcp.Touch{{ID: 1, X: base.X, Y: base.Y}})
	d.Tick(40)
	touch(keyCenter(t, d, "á", true))
	for _, face := range []string{",", "Espacio"} {
		key(face)
	}
	email := fieldRect(1)
	touch(email.Min.Add(email.Size().Div(2)))
	for _, face := range []string{"a", "123", "1", "ABC", "@", "b", ".", "e", "s", "Siguiente"} {
		key(face)
	}
	d.WithGame(func(g ebiten.Game) {
		form := g.(*game)
		if name, email := string(form.values[0]), string(form.values[1]); name != "á, " || email != "a1@b.es" {
			t.Errorf("touch typed Name %q and Email %q, want %q and %q", name, email, "á, ", "a1@b.es")
		}
		if form.active != 2 {
			t.Errorf("Siguiente focused field %d, want Number", form.active)
		}
	})
}
