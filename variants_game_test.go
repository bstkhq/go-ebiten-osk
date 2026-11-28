package osk_test

import (
	"image"
	"slices"
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
)

func spanishGame(t *testing.T) hands {
	t.Helper()
	h := sit(t)
	layout, err := osk.Stock(osk.EsES, osk.Full)
	if err != nil {
		t.Fatal(err)
	}
	h.read(func(g *reader) {
		if err := g.keyboard.SetLayout(layout); err != nil {
			t.Fatal(err)
		}
	})
	h.Tick(2)
	return h
}

func TestSpanishTouchHoldWaitsForASelection(t *testing.T) {
	h := spanishGame(t)
	h.click("a")
	h.settle()
	if got := h.text(); got != "a" {
		t.Fatalf("a short tap typed %q, want a", got)
	}

	at := h.middle("a")
	touch := ebitenmcp.Touch{ID: 8, X: at.X, Y: at.Y}
	inject := h.Runtime().Injector()
	inject.SetTouches([]ebitenmcp.Touch{touch})
	h.Tick(40)
	if got := h.text(); got != "a" {
		t.Fatalf("holding a typed %q before release, want a", got)
	}
	choice := pickerKey(t, h, "á").Rect
	touch.X, touch.Y = choice.Min.X+choice.Dx()/2, choice.Min.Y+choice.Dy()/2
	inject.SetTouches([]ebitenmcp.Touch{touch})
	h.settle()
	if pickerKey(t, h, "á").State != osk.StatePressed {
		t.Fatal("the slid-to variant should be highlighted")
	}
	h.read(func(g *reader) {
		if !slices.Contains(g.taken, osk.PointerID(touch.ID)) || string(g.text) != "a" {
			t.Errorf("sliding left text %q and took %v", g.text, g.taken)
		}
	})
	inject.SetTouches(nil)
	h.settle()
	if got := h.text(); got != "aá" {
		t.Errorf("selecting á typed %q, want aá", got)
	}
	checkPickerClosed(t, h)
}

func TestSpanishMouseHoldCancelsOutsidePicker(t *testing.T) {
	h := spanishGame(t)
	base := h.middle("u")
	inject := h.Runtime().Injector()
	h.Move(float64(base.X), float64(base.Y))
	inject.MouseDown(ebiten.MouseButtonLeft)
	h.Tick(40)
	pickerKey(t, h, "ü")
	h.Move(0, 20)
	inject.MouseUp(ebiten.MouseButtonLeft)
	h.settle()
	if got := h.text(); got != "" {
		t.Errorf("releasing outside typed %q", got)
	}
	checkPickerClosed(t, h)
}

func TestMouseSelectsAVariantWithoutBorrowingItsBasePosition(t *testing.T) {
	h := spanishGame(t)
	base := h.middle("c")
	inject := h.Runtime().Injector()
	h.Move(float64(base.X), float64(base.Y))
	inject.MouseDown(ebiten.MouseButtonLeft)
	h.Tick(40)
	choice := centerOf(pickerKey(t, h, "ç").Rect)
	inject.MoveCursor(float64(choice.X), float64(choice.Y))
	inject.MouseUp(ebiten.MouseButtonLeft)
	h.settle()
	h.read(func(g *reader) {
		want := []ebiten.Key{ebiten.KeyBackslash}
		if string(g.text) != "ç" || !slices.Equal(g.pressed, want) || !slices.Equal(g.released, want) {
			t.Errorf("mouse selection typed %q and gave edges %v / %v, want ç and %v", g.text, g.pressed, g.released, want)
		}
	})
	checkPickerClosed(t, h)
}

func TestATouchTakesOverItsMouseEchosPendingVariant(t *testing.T) {
	h := spanishGame(t)
	base := h.middle("u")
	inject := h.Runtime().Injector()
	h.Move(float64(base.X), float64(base.Y))
	touch := ebitenmcp.Touch{ID: 7, X: base.X, Y: base.Y}
	var game *reader
	h.read(func(g *reader) { game = g })
	h.inOneTick(func() {
		inject.MouseDown(ebiten.MouseButtonLeft)
		game.afterUpdate = func() { inject.SetTouches([]ebitenmcp.Touch{touch}) }
	})
	h.Tick(40)
	at := centerOf(pickerKey(t, h, "ü").Rect)
	touch.X, touch.Y = at.X, at.Y
	inject.SetTouches([]ebitenmcp.Touch{touch})
	h.settle()
	inject.SetTouches(nil)
	h.settle()
	inject.MouseUp(ebiten.MouseButtonLeft)
	h.settle()
	if got := h.text(); got != "ü" {
		t.Errorf("touch with a mouse echo typed %q, want ü once", got)
	}
	checkPickerClosed(t, h)
}

func pickerKey(t *testing.T, h hands, face string) osk.PlacedKey {
	t.Helper()
	var keys []osk.PlacedKey
	h.read(func(g *reader) { keys = g.keyboard.AppendVariants(nil) })
	for _, key := range keys {
		if key.Face() == face {
			return key
		}
	}
	t.Fatalf("picker has %v, want %q", keys, face)
	return osk.PlacedKey{}
}

func checkPickerClosed(t *testing.T, h hands) {
	t.Helper()
	h.read(func(g *reader) {
		if keys := g.keyboard.AppendVariants(nil); len(keys) != 0 {
			t.Errorf("picker still open with %v", keys)
		}
	})
}

func centerOf(rect image.Rectangle) image.Point {
	return rect.Min.Add(rect.Size().Div(2))
}
