package osk_test

import (
	"image"
	"math"
	"testing"

	osk "github.com/bstkhq/go-ebiten-osk"
)

func TestFullTextAndEmailShareTheLetterGrid(t *testing.T) {
	for _, locale := range osk.StockLocales() {
		t.Run(string(locale), func(t *testing.T) {
			for _, rect := range []image.Rectangle{image.Rect(0, 240, 960, 540), image.Rect(7, 3, 504, 242)} {
				text := newKindBoard(t, locale, osk.Full, osk.KindText, osk.ActionGo)
				email := newKindBoard(t, locale, osk.Full, osk.KindEmail, osk.ActionGo)
				text.SetRect(rect)
				email.SetRect(rect)
				pitch := checkFullLetterGrid(t, text, ",")
				checkFullLetterGrid(t, email, "@")
				if locale == osk.EsES {
					text.key(t, "Ir")
					email.key(t, "Ir")
				}

				if comma, at := text.key(t, ",").Rect, email.key(t, "@").Rect; comma != at {
					t.Errorf("comma at %v and @ at %v should occupy the same slot", comma, at)
				}
				if textSpace, emailSpace := roleKey(t, text, osk.RoleSpace), roleKey(t, email, osk.RoleSpace); textSpace != emailSpace {
					t.Errorf("Text Space at %v and Email Space at %v should occupy the same slots", textSpace, emailSpace)
				}
				if locale == osk.EnUS {
					checkFullBoardGap(t, email)
				}
				checkFullLayerWidths(t, text, pitch)
			}
		})
	}
}

func checkFullLetterGrid(t *testing.T, b board, context string) float64 {
	t.Helper()
	keys := b.AppendKeys(nil)
	checkKeyGeometry(t, b, keys)
	first, last := keys[0].Rect, keys[0].Rect
	var count int
	for _, key := range keys {
		if key.Rect.Min.Y != first.Min.Y {
			break
		}
		last = key.Rect
		count++
	}
	pitch := float64(last.Max.X-first.Min.X) / float64(count)
	var controls []image.Rectangle
	for _, key := range keys {
		if key.Role() == osk.RoleControl {
			controls = append(controls, key.Rect)
		}
		checkFullKeyWidth(t, key, pitch, false)
	}

	digits, action := b.key(t, "123").Rect, roleKey(t, b, osk.RoleAction)
	if len(controls) != 2 {
		t.Fatalf("letters layer has %d controls, want Shift and erase", len(controls))
	}
	shift, erase := controls[0], controls[1]
	if digits.Min.X != shift.Min.X || digits.Max.X != shift.Max.X ||
		action.Min.X != erase.Min.X || action.Max.X != erase.Max.X ||
		digits.Min.X != first.Min.X || action.Max.X != last.Max.X {
		t.Errorf("top row from %d to %d, Shift and 123 %v and %v, erase and action %v and %v",
			first.Min.X, last.Max.X, shift, digits, erase, action)
	}

	key, space, dot := b.key(t, context).Rect, roleKey(t, b, osk.RoleSpace), b.key(t, ".").Rect
	if key.Min.Y != space.Min.Y || dot.Min.Y != space.Min.Y || key.Max.X != space.Min.X || space.Max.X != dot.Min.X {
		t.Errorf("want adjacent %s, Space and dot, got %v, %v and %v", context, key, space, dot)
	}
	b.tap(t, 1, context)
	b.Update()
	if b.typed() != context {
		t.Errorf("tapping %q typed %q", context, b.typed())
	}
	return pitch
}

func checkFullLayerWidths(t *testing.T, b board, pitch float64) {
	t.Helper()
	switchLayer(t, b, "123")
	for _, key := range b.AppendKeys(nil) {
		checkFullKeyWidth(t, key, pitch, true)
	}
	for _, key := range b.AppendKeys(nil) {
		if key.Role() == osk.RoleLayerSwitch && key.Face() != "ABC" {
			switchLayer(t, b, key.Face())
			for _, accentKey := range b.AppendKeys(nil) {
				checkFullKeyWidth(t, accentKey, pitch, false)
			}
			break
		}
	}
}

func checkFullKeyWidth(t *testing.T, key osk.PlacedKey, pitch float64, symbols bool) {
	t.Helper()
	want := 1.0
	switch {
	case key.Role() == osk.RoleSpace:
		want = 4
	case symbols && key.Icon == osk.IconErase:
		want = 1
	case key.Role() == osk.RoleControl || key.Role() == osk.RoleAction:
		want = 1.5
	case key.Role() == osk.RoleLayerSwitch && (key.Face() == "ABC" || key.Face() == "123"):
		want = 1.5
	}
	if math.Abs(float64(key.Rect.Dx())-want*pitch) > 1.1 {
		t.Errorf("%q width = %d, want %g slots at pitch %.2f", key.Face(), key.Rect.Dx(), want, pitch)
	}
}

func roleKey(t *testing.T, b board, role osk.Role) image.Rectangle {
	t.Helper()
	for _, key := range b.AppendKeys(nil) {
		if key.Role() == role {
			return key.Rect
		}
	}
	t.Fatalf("the keyboard shows no %v key", role)
	return image.Rectangle{}
}

func checkFullBoardGap(t *testing.T, b board) {
	t.Helper()
	digits, at := b.key(t, "123").Rect, b.key(t, "@").Rect
	if digits.Max.X >= at.Min.X {
		t.Fatalf("want a gap between 123 and @, got %v and %v", digits, at)
	}
	b.touchAt(2, (digits.Max.X+at.Min.X)/2, digits.Min.Y+digits.Dy()/2)
	b.Update()
	if events := b.AppendEvents(nil); len(events) != 0 || !b.PointerTaken(2) {
		t.Errorf("touching the gap gave events %v and taken %t, want no events and taken",
			events, b.PointerTaken(2))
	}
}

func TestFullSpanishEnterShowsTheReturnIcon(t *testing.T) {
	for _, kind := range []osk.Kind{osk.KindText, osk.KindEmail} {
		b := newKindBoard(t, osk.EsES, osk.Full, kind, osk.ActionEnter)
		if icon := b.key(t, "Intro").Icon; icon != osk.IconAction {
			t.Errorf("kind %v shows Intro with icon %v, want the return icon", kind, icon)
		}
		checkLocalizedAction(t, b, osk.ActionEnter, "Intro")
	}
	b := newKindBoard(t, osk.EsES, osk.Full, osk.KindEmail, osk.ActionGo)
	if icon := b.key(t, "Ir").Icon; icon != "" {
		t.Errorf("Ir shows icon %v, want the localized label", icon)
	}
	checkLocalizedAction(t, b, osk.ActionGo, "Ir")
}
