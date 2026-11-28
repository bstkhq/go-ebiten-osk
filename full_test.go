package osk_test

import (
	"image"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestFullSpanishShiftLatchesLocksAndUnlocks(t *testing.T) {
	b := fullBoard(t, osk.EsES)
	shift := func(want osk.State) {
		t.Helper()
		b.tap(t, 1, "Mayús")
		b.Update()
		if got := b.key(t, "Mayús").State; got != want {
			t.Errorf("shift shows %v, want %v", got, want)
		}
		if got := b.AppendEvents(nil); len(got) != 0 {
			t.Errorf("shift reported events %v", got)
		}
		if got := b.input.AppendJustPressedKeys(nil); len(got) != 0 || b.typed() != "" {
			t.Errorf("shift pressed %v and typed %q", got, b.typed())
		}
	}

	shift(osk.StateLatched)
	b.tap(t, 1, ".")
	b.Update()
	if b.typed() != "." || b.key(t, "Mayús").State != osk.StateLatched {
		t.Error("typing punctuation should leave the one capital available")
	}
	b.touch(t, 2, "Ñ")
	b.Update()
	want := []osk.Event{{Kind: osk.KeyDown, Key: ebiten.KeySemicolon}, {Kind: osk.Text, Text: "Ñ"}}
	if got := b.AppendEvents(nil); !slices.Equal(got, want) || b.typed() != "Ñ" {
		t.Errorf("capital typed %q and reported %v, want Ñ and %v", b.typed(), got, want)
	}
	if !b.input.IsKeyJustPressed(ebiten.KeySemicolon) || b.key(t, "ñ").State != osk.StatePressed {
		t.Error("the one capital did not keep its physical position held as the face returned to lower case")
	}
	if got := b.key(t, "Mayús").State; got != osk.StateIdle {
		t.Errorf("shift after one capital is %v, want idle", got)
	}
	b.lift(2)
	b.Update()
	if !b.input.IsKeyJustReleased(ebiten.KeySemicolon) {
		t.Error("releasing the capital did not release semicolon")
	}

	shift(osk.StateLatched)
	shift(osk.StateLocked)
	for _, face := range []string{"A", "Ñ"} {
		b.tap(t, 1, face)
		b.Update()
		if b.typed() != face || b.key(t, "Mayús").State != osk.StateLocked {
			t.Errorf("with caps locked, tapping %s typed %q", face, b.typed())
		}
	}
	shift(osk.StateIdle)
	b.key(t, "ñ")
}

func fullBoard(t *testing.T, locale osk.Locale) board {
	t.Helper()
	layout, err := osk.Stock(locale, osk.Full)
	if err != nil {
		t.Fatal(err)
	}
	return newBoard(t, layout)
}

func TestFullLayersTypeAndReturnToLetters(t *testing.T) {
	for _, test := range []struct {
		locale                     osk.Locale
		letters, symbols           []string
		accents                    []string
		accentLabel                string
		shift, erase, space, enter string
		positions                  map[rune]ebiten.Key
	}{
		{
			locale: osk.EnUS, letters: []string{"qwertyuiop", "asdfghjkl", "zxcvbnm"},
			symbols: []string{"1234567890", ",-_'?!@:/&", "`=[]\\;()\""},
			shift:   "Shift", erase: "Delete", space: "Space", enter: "Enter",
			positions: map[rune]ebiten.Key{
				'.': ebiten.KeyPeriod, ',': ebiten.KeyComma, '-': ebiten.KeyMinus,
				'\'': ebiten.KeyQuote, '/': ebiten.KeySlash, '`': ebiten.KeyBackquote,
				'=': ebiten.KeyEqual, '[': ebiten.KeyBracketLeft, ']': ebiten.KeyBracketRight,
				'\\': ebiten.KeyBackslash, ';': ebiten.KeySemicolon,
			},
		},
		{
			locale: osk.EsES, letters: []string{"qwertyuiop", "asdfghjklñ", "zxcvbnm"},
			accents: []string{"áéí", "óú", "üç"}, accentLabel: "ÁÉÍ",
			symbols: []string{"1234567890", ",-_'¿?¡!@+", "`´:;()/º<"},
			shift:   "Mayús", erase: "Borrar", space: "Espacio", enter: "Intro",
			positions: map[rune]ebiten.Key{
				'ñ': ebiten.KeySemicolon, 'Ñ': ebiten.KeySemicolon,
				'ç': ebiten.KeyBackslash, 'Ç': ebiten.KeyBackslash, 'º': ebiten.KeyBackquote,
				'<': ebiten.KeyIntlBackslash, '\'': ebiten.KeyMinus, '¡': ebiten.KeyEqual,
				'`': ebiten.KeyBracketLeft, '´': ebiten.KeyQuote, '+': ebiten.KeyBracketRight,
				',': ebiten.KeyComma, '.': ebiten.KeyPeriod, '-': ebiten.KeySlash,
			},
		},
		{
			locale: "en-GB", letters: []string{"qwertyuiop", "asdfghjkl", "zxcvbnm"},
			symbols: []string{"1234567890", ",-_'?!@:/&£", "`=[]\\;()\"#"},
			shift:   "Shift", erase: "Backspace", space: "Space", enter: "Return", positions: britishPositions,
		},
		{
			locale: "de-DE", letters: []string{"qwertzuiopü", "asdfghjklöä", "yxcvbnmß"},
			symbols: []string{"1234567890", ",-_'?!@:/&", "+#<>;()\"="},
			shift:   "Umschalt", erase: "Löschen", space: "Leerzeichen", enter: "Eingabe", positions: germanPositions,
		},
		{
			locale: "it-IT", letters: []string{"qwertyuiop", "asdfghjkl", "zxcvbnm"},
			symbols: []string{"1234567890", ",-_'?!@:/&", "\\+<>;()\"£"},
			accents: []string{"àèé", "ìò", "ù"}, accentLabel: "ÀÈÉ",
			shift: "Maiusc", erase: "Cancella", space: "Spazio", enter: "Invio", positions: italianPositions,
		},
	} {
		t.Run(string(test.locale), func(t *testing.T) {
			b := fullBoard(t, test.locale)
			b.SetRect(image.Rect(7, 3, 1004, 398))
			home := b.key(t, "123").Rect
			bottom := []string{"123", test.space, ".", test.enter}
			if test.accentLabel != "" {
				bottom = slices.Insert(bottom, 1, test.accentLabel)
			}
			checkFullLayer(t, b, fullRows(test.letters, bottom, test.shift, test.erase), test.space, test.erase, test.positions)

			switchLayer(t, b, "123")
			bottom[0] = "ABC"
			checkFullLayer(t, b, fullRows(test.symbols, bottom, "", test.erase), test.space, test.erase, test.positions)
			checkFullReturn(t, b, home, "q")
			if test.accentLabel != "" {
				switchLayer(t, b, test.accentLabel)
				bottom[1] = "123"
				checkFullLayer(t, b, fullRows(test.accents, bottom, test.shift, test.erase), test.space, test.erase, test.positions)
				for range 2 {
					b.tap(t, 1, test.shift)
					b.Update()
				}
				capitals := []string{strings.ToUpper(test.accents[0]), strings.ToUpper(test.accents[1]), strings.ToUpper(test.accents[2])}
				checkFullLayer(t, b, fullRows(capitals, bottom, test.shift, test.erase), test.space, test.erase, test.positions)
				checkFullReturn(t, b, home, "Q") // Caps lock persists across layers.
			}
		})
	}
}

func checkFullReturn(t *testing.T, b board, home image.Rectangle, face string) {
	t.Helper()
	if got := b.key(t, "ABC").Rect; got != home {
		t.Errorf("the way back to letters is at %v, want %v", got, home)
	}
	switchLayer(t, b, "ABC")
	b.key(t, face)
}

func fullRows(chars, bottom []string, shift, erase string) [][]string {
	third := strings.Split(chars[2], "")
	if shift != "" {
		third = append([]string{shift}, third...)
	}
	return [][]string{strings.Split(chars[0], ""), strings.Split(chars[1], ""), append(third, erase), bottom}
}

func checkFullLayer(t *testing.T, b board, rows [][]string, space, erase string, positions map[rune]ebiten.Key) {
	t.Helper()
	keys := b.AppendKeys(nil)
	checkKeyGeometry(t, b, keys)
	want := slices.Concat(rows...)
	if len(keys) != len(want) {
		t.Fatalf("the layer has %d keys, want %d", len(keys), len(want))
	}
	for row, faces := range rows {
		top := b.Rect().Min.Y + row*b.Rect().Dy()/len(rows)
		for _, face := range faces {
			key := keys[0]
			keys = keys[1:]
			if key.Face() != face || key.Rect.Min.Y != top {
				t.Fatalf("row %d has key %q at y=%d, want %q at y=%d", row, key.Face(), key.Rect.Min.Y, face, top)
			}
			if face == space || face == erase || utf8.RuneCountInString(face) == 1 {
				checkTyping(t, b, face, []string{space, erase}, positions)
			}
		}
	}
}

func TestChangingFormReleasesHoldsAndClearsShift(t *testing.T) {
	b := fullBoard(t, osk.EnUS)
	for range 2 {
		b.tap(t, 1, "Shift")
		b.Update()
	}
	b.touch(t, 2, "Q")
	b.Update()
	if err := b.SetLayout(stockBoard(t, osk.EnUS)); err != nil {
		t.Fatal(err)
	}
	b.Update()
	checkRelease(t, b, ebiten.KeyQ)
	layout, err := osk.Stock(osk.EnUS, osk.Full)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.SetLayout(layout); err != nil {
		t.Fatal(err)
	}
	if !b.Shown() || b.key(t, "Shift").State != osk.StateIdle {
		t.Fatal("the replacement should be visible with shift idle")
	}
	b.tap(t, 3, "q")
	b.Update()
	if b.typed() != "q" {
		t.Errorf("the replacement typed %q, want q", b.typed())
	}
}

func TestUnknownFieldActionsAreRejected(t *testing.T) {
	const unknown = osk.FieldAction(255)
	if _, err := osk.StockWithAction(osk.EnUS, osk.Full, unknown); err == nil {
		t.Error("accepted an unknown stock field action")
	}
	layout := osk.Layout{Layers: []osk.Layer{{Rows: [][]osk.Key{{{Do: osk.Submit(unknown)}}}}}}
	if _, err := osk.New(layout); err == nil {
		t.Error("accepted an unknown custom field action")
	}
}

func TestFullActionLabelsAndEvents(t *testing.T) {
	for _, test := range []struct {
		action             osk.FieldAction
		en, es, gb, de, it string
	}{
		{osk.ActionEnter, "Enter", "Intro", "Return", "Eingabe", "Invio"},
		{osk.ActionNext, "Next", "Siguiente", "Next", "Weiter", "Avanti"},
		{osk.ActionDone, "Done", "Listo", "Done", "Fertig", "Fine"},
		{osk.ActionGo, "Go", "Ir", "Go", "Los", "Vai"},
		{osk.ActionSearch, "Search", "Buscar", "Search", "Suchen", "Cerca"},
		{osk.ActionSend, "Send", "Enviar", "Send", "Senden", "Invia"},
	} {
		for locale, label := range map[osk.Locale]string{
			osk.EnUS: test.en, osk.EsES: test.es, "en-GB": test.gb, "de-DE": test.de, "it-IT": test.it,
		} {
			t.Run(string(locale)+"/"+label, func(t *testing.T) {
				layout, err := osk.StockWithAction(locale, osk.Full, test.action)
				if err != nil {
					t.Fatal(err)
				}
				b := newBoard(t, layout)
				if got := b.key(t, label).Role(); got != osk.RoleAction {
					t.Errorf("%q has role %v, want action", label, got)
				}
				b.touch(t, 1, label)
				b.Update()
				want := []osk.Event{
					{Kind: osk.KeyDown, Key: ebiten.KeyEnter},
					{Kind: osk.ActionPressed, FieldAction: test.action},
				}
				if got := b.AppendEvents(nil); !slices.Equal(got, want) {
					t.Errorf("tapping %q reported %v, want %v", label, got, want)
				} else if formatted, label := got[1].String(), "action pressed "+strings.ToLower(test.en); formatted != label {
					t.Errorf("the action event displays %q, want %q", formatted, label)
				}
				if !b.input.IsKeyJustPressed(ebiten.KeyEnter) || b.typed() != "" {
					t.Errorf("tapping %q pressed %v and typed %q", label, b.input.AppendJustPressedKeys(nil), b.typed())
				}
				b.Update()
				if !b.input.IsKeyPressed(ebiten.KeyEnter) || len(b.AppendEvents(nil)) != 0 {
					t.Error("holding the action key should keep Enter down silently")
				}
				b.lift(1)
				b.Update()
				if !b.input.IsKeyJustReleased(ebiten.KeyEnter) {
					t.Error("lifting the action key should release Enter")
				}
			})
		}
	}
}
