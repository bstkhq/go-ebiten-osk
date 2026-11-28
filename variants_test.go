package osk_test

import (
	"image"
	"slices"
	"strings"
	"testing"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestRemovingSpanishAccentSwitchKeepsTheFullSpaceBounds(t *testing.T) {
	for _, test := range []struct {
		kind    osk.Kind
		context string
	}{{osk.KindText, ","}, {osk.KindEmail, "@"}} {
		b := newKindBoard(t, osk.EsES, osk.Full, test.kind, osk.ActionGo)
		b.SetRect(image.Rect(0, 240, 960, 540))
		for face, rect := range map[string]image.Rectangle{
			"123":        image.Rect(0, 465, 144, 540),
			"Mayús":      image.Rect(0, 390, 144, 465),
			test.context: image.Rect(240, 465, 336, 540),
			"Espacio":    image.Rect(336, 465, 720, 540),
			".":          image.Rect(720, 465, 816, 540),
		} {
			if got := b.key(t, face).Rect; got != rect {
				t.Errorf("key %q occupies %v, want %v", face, got, rect)
			}
		}
		b.touchAt(1, 190, 500)
		b.Update()
		if !b.PointerTaken(1) || len(b.AppendEvents(nil)) != 0 {
			t.Errorf("the former accent slot took %t and emitted %v", b.PointerTaken(1), b.AppendEvents(nil))
		}
	}
}

func TestStockPickersKeepEachLocalesRepertoireAndPositions(t *testing.T) {
	for _, test := range []struct {
		locale    osk.Locale
		shift     string
		groups    []string
		positions map[rune]ebiten.Key
	}{
		{osk.EsES, "Mayús", []string{"aá", "eé", "ií", "oó", "uúü", "cç"},
			map[rune]ebiten.Key{'ç': ebiten.KeyBackslash, 'Ç': ebiten.KeyBackslash}},
		{osk.CaES, "Majús", []string{"aà", "eèé", "iíï", "oòó", "uúü"}, nil},
		{osk.FrFR, "Maj", []string{"aàâ", "cç", "eéèêë", "iîï", "oôœ", "uùû"}, localeCases[0].positions},
		{osk.ItIT, "Maiusc", []string{"aà", "eèé", "iì", "oò", "uù"}, italianPositions},
		{osk.PtPT, "Maiús", []string{"aáàâã", "eéê", "ií", "oóôõ", "uú"}, nil},
	} {
		t.Run(string(test.locale), func(t *testing.T) {
			for _, form := range []osk.Form{osk.Compact, osk.Full} {
				b := newKindBoard(t, test.locale, form, osk.KindText, osk.ActionEnter)
				for _, group := range test.groups {
					if form == osk.Compact {
						group = strings.ToUpper(group)
					}
					checkVariantGroup(t, b, group, test.positions)
				}
				if form == osk.Full {
					b.tap(t, 1, test.shift)
					b.Update()
					group := []rune(strings.ToUpper(test.groups[0]))
					checkVariant(t, b, group, group[1], test.positions)
					if state := b.key(t, test.shift).State; state != osk.StateIdle {
						t.Errorf("after selecting a capital, Shift is %v, want idle", state)
					}
					for range 2 {
						b.tap(t, 1, test.shift)
						b.Update()
					}
					checkVariantGroup(t, b, strings.ToUpper(test.groups[1]), test.positions)
					if state := b.key(t, test.shift).State; state != osk.StateLocked {
						t.Errorf("after selecting with caps lock, Shift is %v, want locked", state)
					}
				}
			}
		})
	}
}

func checkVariantGroup(t *testing.T, b board, group string, positions map[rune]ebiten.Key) {
	t.Helper()
	runes := []rune(group)
	for _, variant := range runes[1:] {
		checkVariant(t, b, runes, variant, positions)
	}
}

func checkVariant(t *testing.T, b board, group []rune, variant rune, positions map[rune]ebiten.Key) {
	t.Helper()
	holdLetter(t, b, string(group[0]))
	keys := b.AppendVariants(nil)
	var faces []string
	for _, key := range keys {
		faces = append(faces, key.Face())
	}
	if want := strings.Split(string(group[1:]), ""); !slices.Equal(faces, want) {
		t.Fatalf("holding %c offered %v, want %v", group[0], faces, want)
	}
	at := centerOf(keys[slices.Index(faces, string(variant))].Rect)
	b.lift(1)
	b.touchAt(1, at.X, at.Y)
	b.Update()
	b.lift(1)
	b.Update()
	var want []ebiten.Key
	if position, ok := positions[variant]; ok {
		want = []ebiten.Key{position}
	}
	if got := b.input.AppendJustPressedKeys(nil); b.typed() != string(variant) || !slices.Equal(got, want) {
		t.Errorf("selecting %c typed %q and pressed %v, want %v", variant, b.typed(), got, want)
	}
}

func holdLetter(t *testing.T, b board, letter string) {
	t.Helper()
	b.touch(t, 1, letter)
	for range 35 {
		b.Update()
		if b.typed() != "" {
			t.Fatalf("holding %s typed %q", letter, b.typed())
		}
	}
}

func TestChangingTheBoardCancelsAPendingVariant(t *testing.T) {
	for _, action := range []string{"hide", "field", "layer", "resize", "missed tick"} {
		t.Run(action, func(t *testing.T) {
			b := fullBoard(t, osk.EsES)
			holdLetter(t, b, "a")
			picker := b.AppendVariants(nil)
			if len(picker) != 1 {
				t.Fatalf("picker has %d choices, want 1", len(picker))
			}
			at := centerOf(picker[0].Rect)
			b.lift(1)
			b.touchAt(1, at.X, at.Y)
			b.Update()

			switch action {
			case "hide":
				b.Hide()
				b.Show()
			case "field":
				layout, err := osk.StockWithKind(osk.EsES, osk.Full, osk.KindEmail, osk.ActionNext)
				if err != nil {
					t.Fatal(err)
				}
				if err := b.SetLayout(layout); err != nil {
					t.Fatal(err)
				}
			case "layer":
				b.tap(t, 2, "123")
				b.Update()
			case "resize":
				b.SetRect(b.Rect().Inset(5))
			case "missed tick":
				b.skip()
			}
			b.lift(1)
			b.Update()
			if b.typed() != "" || len(b.AppendVariants(nil)) != 0 {
				t.Errorf("after %s, typed %q and still shows %v", action, b.typed(), b.AppendVariants(nil))
			}
		})
	}
}

func TestAnOpenPickerKeepsOtherKeysResponsive(t *testing.T) {
	b := fullBoard(t, osk.EsES)
	holdLetter(t, b, "a")
	b.touch(t, 2, "b")
	b.Update()
	if b.typed() != "b" || !b.input.IsKeyPressed(ebiten.KeyB) || !b.PointerTaken(1) || !b.PointerTaken(2) {
		t.Errorf("with the picker open, the second finger typed %q and held %v", b.typed(), b.input.AppendPressedKeys(nil))
	}
	b.lift(2)
	b.Update()
	b.lift(1)
	b.Update()
	if b.typed() != "" || len(b.AppendVariants(nil)) != 0 {
		t.Errorf("lifting on the original letter typed %q and left picker %v", b.typed(), b.AppendVariants(nil))
	}
}

func TestVariantTapsRollOverInTheOrderKeysAreTouched(t *testing.T) {
	for _, test := range []struct{ second, want string }{
		{"K", "Ok"}, {"Espacio", "O "}, {"123", "O"}, {"A", "Oa"},
	} {
		t.Run(test.second, func(t *testing.T) {
			b := fullBoard(t, osk.EsES)
			switchLayer(t, b, "Mayús")
			b.touch(t, 1, "O")
			b.Update()
			b.touch(t, 2, test.second)
			b.Update()
			got := b.typed()
			b.lift(1)
			b.lift(2)
			b.Update()
			got += b.typed()
			if got != test.want || len(b.AppendVariants(nil)) != 0 {
				t.Errorf("rolling from Shift and o to %q typed %q, want %q", test.second, got, test.want)
			}
		})
	}
}

func TestSlidingOffBeforeThePickerFreesTheLetterForAnotherTap(t *testing.T) {
	b := fullBoard(t, osk.EsES)
	b.touch(t, 1, "a")
	b.Update()
	at := centerOf(b.key(t, "s").Rect)
	b.lift(1)
	b.touchAt(1, at.X, at.Y)
	b.Update()
	if state := b.key(t, "a").State; state != osk.StateIdle {
		t.Errorf("the abandoned letter shows %v, want idle", state)
	}
	b.tap(t, 2, "a")
	b.Update()
	if got := b.typed(); got != "a" {
		t.Errorf("the second finger typed %q, want a", got)
	}
	b.lift(1)
	b.Update()
	if got := b.typed(); got != "" {
		t.Errorf("the abandoned finger typed %q", got)
	}
}
