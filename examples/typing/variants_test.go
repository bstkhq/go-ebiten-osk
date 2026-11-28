package main

import (
	"image"
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/bstkhq/go-ebiten-osk/paint"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestTypingDemoSelectsAndCancelsSpanishVariants(t *testing.T) {
	d := ebitenmcp.T(t)
	exerciseSpanishPicker(t, d)
}

func exerciseSpanishPicker(t *testing.T, d *ebitenmcp.Driver) {
	t.Helper()
	d.Tap(ebiten.KeyF1) // Spanish is next after US English.
	d.Tap(ebiten.KeyF3) // Full.
	touch := func(at image.Point) {
		d.Runtime().Injector().SetTouches([]ebitenmcp.Touch{{ID: 1, X: at.X, Y: at.Y}})
		d.Tick(2)
	}
	release := func() {
		d.Runtime().Injector().SetTouches(nil)
		d.Tick(2)
	}
	for _, letter := range []string{"c", "a", "m", "i"} {
		touch(keyCenter(t, d, letter, false))
		release()
	}
	touch(keyCenter(t, d, "o", false))
	d.Tick(40)
	choice := keyCenter(t, d, "ó", true)
	touch(choice)
	assertPickerPainted(t, d, choice)
	release()
	touch(keyCenter(t, d, "n", false))
	release()

	touch(keyCenter(t, d, "u", false))
	d.Tick(40)
	touch(keyCenter(t, d, "ü", true))
	touch(image.Pt(30, 190))
	release()
	d.WithGame(func(current ebiten.Game) {
		form := current.(*game)
		if got := string(form.values[0]); got != "camión" {
			t.Errorf("selecting ó and cancelling ü left %q, want camión", got)
		}
		if keys := form.keyboard.AppendVariants(nil); len(keys) != 0 {
			t.Errorf("the cancelled picker still has %v", keys)
		}
	})
}

func assertPickerPainted(t *testing.T, d *ebitenmcp.Driver, choice image.Point) {
	t.Helper()
	screen := d.Screenshot()
	pressed := paint.Dark().Colors(osk.RoleCharacter, osk.StatePressed).Face
	if at := choice.Sub(image.Pt(20, 0)); screen.RGBAAt(at.X, at.Y) != pressed {
		t.Errorf("the selected alternative at %v did not paint its pressed face", at)
	}
}

func keyCenter(t *testing.T, d *ebitenmcp.Driver, face string, picker bool) image.Point {
	t.Helper()
	var keys []osk.PlacedKey
	d.WithGame(func(current ebiten.Game) {
		keyboard := current.(*game).keyboard
		if picker {
			keys = keyboard.AppendVariants(nil)
		} else {
			keys = keyboard.AppendKeys(nil)
		}
	})
	for _, key := range keys {
		if key.Face() == face {
			return key.Rect.Min.Add(key.Rect.Size().Div(2))
		}
	}
	t.Fatalf("no key %q among %v", face, keys)
	return image.Point{}
}
