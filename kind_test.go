package osk_test

import (
	"slices"
	"testing"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestStockTextKindKeepsTheExistingLayouts(t *testing.T) {
	for _, locale := range []osk.Locale{osk.EnUS, osk.EsES} {
		for _, form := range []osk.Form{osk.Compact, osk.Full} {
			t.Run(string(locale)+"/"+form.String(), func(t *testing.T) {
				layout, err := osk.StockWithAction(locale, form, osk.ActionDone)
				if err != nil {
					t.Fatal(err)
				}
				want := newBoard(t, layout).AppendKeys(nil)
				got := newKindBoard(t, locale, form, osk.KindText, osk.ActionDone).AppendKeys(nil)
				if !slices.Equal(got, want) {
					t.Error("text kind changed the stock keys")
				}
			})
		}
	}
}

func TestStockEmailKeys(t *testing.T) {
	for _, test := range []struct {
		locale osk.Locale
		space  string
	}{{osk.EnUS, "Space"}, {osk.EsES, "Espacio"}} {
		t.Run(string(test.locale), func(t *testing.T) {
			b := newKindBoard(t, test.locale, osk.Full, osk.KindEmail, osk.ActionNext)
			at, space, dot := b.key(t, "@").Rect, b.key(t, test.space).Rect, b.key(t, ".").Rect
			if at.Min.Y != space.Min.Y || at.Max.X != space.Min.X || space.Max.X != dot.Min.X {
				t.Fatalf("the letters layer should place @, space and dot beside each other")
			}
			for _, key := range []struct {
				face string
				want []ebiten.Key
			}{{"q", []ebiten.Key{ebiten.KeyQ}}, {"@", nil}, {".", []ebiten.Key{ebiten.KeyPeriod}}} {
				b.tap(t, 1, key.face)
				b.Update()
				if got := b.input.AppendJustPressedKeys(nil); b.typed() != key.face || !slices.Equal(got, key.want) {
					t.Errorf("tapping %q typed %q and pressed %v, want %v", key.face, b.typed(), got, key.want)
				}
			}
			compact := newKindBoard(t, test.locale, osk.Compact, osk.KindEmail, osk.ActionNext)
			compact.key(t, "Q")
			switchLayer(t, compact, "123")
			compact.key(t, "@")
		})
	}
}

func TestStockNumberKeys(t *testing.T) {
	for _, test := range []struct {
		locale      osk.Locale
		erase, done string
	}{{osk.EnUS, "Delete", "Done"}, {osk.EsES, "Borrar", "Listo"}} {
		for _, form := range []osk.Form{osk.Compact, osk.Full} {
			t.Run(string(test.locale)+"/"+form.String(), func(t *testing.T) {
				b := newKindBoard(t, test.locale, form, osk.KindNumber, osk.ActionDone)
				keys := b.AppendKeys(nil)
				checkKeyGeometry(t, b, keys)
				if len(keys) != 12 {
					t.Fatalf("number pad has %d keys, want 12", len(keys))
				}
				for digit := '0'; digit <= '9'; digit++ {
					b.tap(t, 1, string(digit))
					b.Update()
					want := ebiten.KeyDigit0 + ebiten.Key(digit-'0')
					if got := b.input.AppendJustPressedKeys(nil); b.typed() != string(digit) || !slices.Equal(got, []ebiten.Key{want}) {
						t.Errorf("tapping %c typed %q and pressed %v, want %v", digit, b.typed(), got, want)
					}
				}
				b.tap(t, 1, test.erase)
				b.Update()
				if !b.input.IsKeyJustPressed(ebiten.KeyBackspace) || b.typed() != "" {
					t.Error("erase should press Backspace without typing")
				}
				b.touch(t, 2, test.done)
				b.Update()
				want := osk.Event{Kind: osk.ActionPressed, FieldAction: osk.ActionDone}
				if !b.input.IsKeyJustPressed(ebiten.KeyEnter) || !slices.Contains(b.AppendEvents(nil), want) {
					t.Error("the action should press Enter and report Done")
				}
			})
		}
	}
}

func TestChangingKindWhileShown(t *testing.T) {
	b := newKindBoard(t, osk.EnUS, osk.Full, osk.KindText, osk.ActionNext)
	for range 2 {
		b.tap(t, 1, "Shift")
		b.Update()
	}
	b.touch(t, 2, "Q")
	b.Update()
	for _, test := range []struct {
		kind osk.Kind
		face string
	}{{osk.KindEmail, "q"}, {osk.KindNumber, "1"}, {osk.KindText, "q"}} {
		layout, err := osk.StockWithKind(osk.EnUS, osk.Full, test.kind, osk.ActionNext)
		if err != nil {
			t.Fatal(err)
		}
		if err := b.SetLayout(layout); err != nil {
			t.Fatal(err)
		}
		if !b.Shown() || b.key(t, test.face).State != osk.StateIdle {
			t.Fatalf("switching kind should show the new layout immediately")
		}
		b.Update()
		if test.kind == osk.KindEmail {
			checkRelease(t, b, ebiten.KeyQ)
		}
		b.tap(t, 1, test.face)
		b.Update()
		if got := b.typed(); got != test.face {
			t.Errorf("the replacement typed %q, want %q", got, test.face)
		}
	}
}

func TestStockKindsRejectUnknownSelections(t *testing.T) {
	for _, test := range []struct {
		locale osk.Locale
		form   osk.Form
		kind   osk.Kind
		action osk.FieldAction
	}{
		{osk.EnUS, osk.Full, osk.Kind(255), osk.ActionEnter},
		{"unknown", osk.Compact, osk.KindNumber, osk.ActionEnter},
		{osk.EnUS, osk.Form(255), osk.KindNumber, osk.ActionEnter},
		{osk.EsES, osk.Compact, osk.KindNumber, osk.FieldAction(255)},
	} {
		if _, err := osk.StockWithKind(test.locale, test.form, test.kind, test.action); err == nil {
			t.Errorf("accepted unknown stock selection %+v", test)
		}
	}
}

func newKindBoard(t *testing.T, locale osk.Locale, form osk.Form, kind osk.Kind, action osk.FieldAction) board {
	t.Helper()
	layout, err := osk.StockWithKind(locale, form, kind, action)
	if err != nil {
		t.Fatal(err)
	}
	return newBoard(t, layout)
}
