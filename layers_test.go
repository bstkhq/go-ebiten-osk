package osk_test

import (
	"image"
	"slices"
	"strings"
	"testing"
	"unicode"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestCompactSpanishLettersMatchTheReviewedBoard(t *testing.T) {
	layout, err := osk.Stock("es-ES", osk.Compact)
	if err != nil {
		t.Fatal(err)
	}

	b := newBoard(t, layout)
	rows := []string{"QWERTYUIOP", "ASDFGHJKLÑ", "ZXCVBNM"}
	for y, row := range rows {
		letters := []rune(row)
		x := (10 - len(letters)) * 50
		for _, letter := range letters {
			top := 100 + y*100
			want := image.Rect(x, top, x+100, top+100)
			if got := b.key(t, string(letter)).Rect; got != want {
				t.Errorf("key %q is at %v, want %v", letter, got, want)
			}
			x += 100
		}
	}

	controls := []struct {
		face        string
		left, right int
	}{
		{"123", 0, 200}, {"ÁÉÍ", 200, 400}, {"Espacio", 400, 800}, {"Borrar", 800, 1000},
	}
	for _, key := range controls {
		want := image.Rect(key.left, 400, key.right, 500)
		if got := b.key(t, key.face).Rect; got != want {
			t.Errorf("key %q is at %v, want %v", key.face, got, want)
		}
	}

	if got := len(b.AppendKeys(nil)); got != 31 {
		t.Errorf("the board has %d keys, want exactly 31", got)
	}
}

func TestCompactLayersTypeAndReturnToLetters(t *testing.T) {
	for _, test := range []struct {
		locale               osk.Locale
		space, erase         string
		letters, punctuation string
		auxiliary            []string
		positions            map[rune]ebiten.Key
	}{
		{
			locale: osk.EnUS, space: "Space", erase: "Delete",
			letters: "QWERTYUIOPASDFGHJKLZXCVBNM", punctuation: ".,-_'?!@:/`=[]\\;()&\"",
			auxiliary: []string{"123"},
			positions: map[rune]ebiten.Key{
				'.': ebiten.KeyPeriod, ',': ebiten.KeyComma, '-': ebiten.KeyMinus,
				'\'': ebiten.KeyQuote, '/': ebiten.KeySlash, '`': ebiten.KeyBackquote,
				'=': ebiten.KeyEqual, '[': ebiten.KeyBracketLeft, ']': ebiten.KeyBracketRight,
				'\\': ebiten.KeyBackslash, ';': ebiten.KeySemicolon,
			},
		},
		{
			locale: osk.EsES, space: "Espacio", erase: "Borrar",
			letters: "QWERTYUIOPASDFGHJKLÑZXCVBNM", punctuation: ".,-_'¿?¡!@`+´:;()/º<",
			auxiliary: []string{"123", "ÁÉÍ"},
			positions: map[rune]ebiten.Key{
				'Ñ': ebiten.KeySemicolon, 'Ç': ebiten.KeyBackslash, 'º': ebiten.KeyBackquote,
				'<': ebiten.KeyIntlBackslash, '\'': ebiten.KeyMinus, '¡': ebiten.KeyEqual,
				'`': ebiten.KeyBracketLeft, '´': ebiten.KeyQuote, '+': ebiten.KeyBracketRight, ',': ebiten.KeyComma,
				'.': ebiten.KeyPeriod, '-': ebiten.KeySlash,
			},
		},
	} {
		t.Run(string(test.locale), func(t *testing.T) {
			b := newBoard(t, stockBoard(t, test.locale))
			b.SetRect(image.Rect(7, 3, 1004, 398))
			home := b.key(t, "123").Rect
			controls := []string{test.space, test.erase}
			checkLayer(t, b, test.letters, controls, test.auxiliary, test.positions)

			for i, entry := range test.auxiliary {
				t.Run(entry, func(t *testing.T) {
					switchLayer(t, b, entry)
					if got := b.key(t, "ABC").Rect; got != home {
						t.Errorf("the way back to letters is at %v, want %v", got, home)
					}
					characters := "1234567890" + test.punctuation
					if entry == "ÁÉÍ" {
						characters = "ÁÉÍÓÚÜÇ"
					}
					switches := slices.Clone(test.auxiliary)
					switches[i] = "ABC"
					checkLayer(t, b, characters, controls, switches, test.positions)
					switchLayer(t, b, "ABC")
					if got := b.key(t, "Q").Face(); got != "Q" {
						t.Errorf("after returning to letters, the first key shows %q", got)
					}
				})
			}
		})
	}
}

func stockBoard(t *testing.T, locale osk.Locale) osk.Layout {
	t.Helper()
	layout, err := osk.Stock(locale, osk.Compact)
	if err != nil {
		t.Fatal(err)
	}
	return layout
}

func checkLayer(t *testing.T, b board, characters string, controls, switches []string, positions map[rune]ebiten.Key) {
	t.Helper()
	keys := b.AppendKeys(nil)
	remaining := append(strings.Split(characters, ""), controls...)
	remaining = append(remaining, switches...)

	checkKeyGeometry(t, b, keys)
	for _, key := range keys {
		face := key.Face()
		found := slices.Index(remaining, face)
		if found < 0 {
			t.Errorf("unexpected key %q", face)
			continue
		}
		remaining = slices.Delete(remaining, found, found+1)
		if slices.Contains(switches, face) {
			if key.Role() != osk.RoleLayerSwitch {
				t.Errorf("key %q has role %v, want layer switch", face, key.Role())
			}
			continue
		}
		checkTyping(t, b, face, controls, positions)
	}
	if len(remaining) != 0 {
		t.Errorf("layer is missing the keys %q", remaining)
	}
}

func checkTyping(t *testing.T, b board, face string, controls []string, positions map[rune]ebiten.Key) {
	t.Helper()
	text, pressed := expectedKey(face, controls, positions)
	b.tap(t, 1, face)
	b.Update()
	if got := b.typed(); got != text {
		t.Errorf("key %q typed %q, want %q", face, got, text)
	}
	if got := b.input.AppendJustPressedKeys(nil); !slices.Equal(got, pressed) {
		t.Errorf("key %q pressed %v, want %v", face, got, pressed)
	}
	if got := b.input.AppendJustReleasedKeys(nil); !slices.Equal(got, pressed) {
		t.Errorf("key %q released %v, want %v", face, got, pressed)
	}
}

func checkKeyGeometry(t *testing.T, b board, keys []osk.PlacedKey) {
	t.Helper()
	for i, key := range keys {
		if key.Rect.Empty() || !key.Rect.In(b.Rect()) {
			t.Errorf("key %q is at %v, outside %v", key.Face(), key.Rect, b.Rect())
		}
		for _, other := range keys[i+1:] {
			if key.Rect.Overlaps(other.Rect) {
				t.Errorf("keys %q and %q overlap at %v and %v", key.Face(), other.Face(), key.Rect, other.Rect)
			}
		}
	}
}

func expectedKey(face string, controls []string, positions map[rune]ebiten.Key) (string, []ebiten.Key) {
	switch face {
	case controls[0]:
		return " ", []ebiten.Key{ebiten.KeySpace}
	case controls[1]:
		return "", []ebiten.Key{ebiten.KeyBackspace}
	}
	character := []rune(face)[0]
	capital := unicode.ToUpper(character)
	switch {
	case 'A' <= capital && capital <= 'Z':
		return face, []ebiten.Key{ebiten.KeyA + ebiten.Key(capital-'A')}
	case '0' <= character && character <= '9':
		return face, []ebiten.Key{ebiten.KeyDigit0 + ebiten.Key(character-'0')}
	}
	if position, ok := positions[character]; ok {
		return face, []ebiten.Key{position}
	}
	return face, nil
}

func switchLayer(t *testing.T, b board, face string) {
	t.Helper()
	b.tap(t, 1, face)
	b.Update()
	if events := b.AppendEvents(nil); len(events) != 0 {
		t.Errorf("switch %q reported events %v", face, events)
	}
	if pressed := b.input.AppendPressedKeys(nil); len(pressed) != 0 || b.typed() != "" {
		t.Errorf("switch %q pressed %v and typed %q", face, pressed, b.typed())
	}
}

func TestSwitchingLayersReleasesOldHoldsAndKeepsPointersDetached(t *testing.T) {
	b := newBoard(t, stockBoard(t, osk.EsES))
	b.touch(t, 1, "Q")
	b.touch(t, 2, "Q")
	b.Update()

	b.touch(t, 3, "123")
	b.touch(t, 4, "ÁÉÍ")
	b.Update()
	b.key(t, "1") // Both landed on the letters. The first switch won.
	checkRelease(t, b, ebiten.KeyQ)
	for _, id := range []osk.PointerID{1, 2, 3, 4} {
		if !b.PointerTaken(id) {
			t.Errorf("the layer switch gave up pointer %d before lift", id)
		}
	}

	b.lift(2)
	b.reland(t, 1, "1")
	b.Update()
	if b.PointerTaken(2) {
		t.Error("the pointer stayed taken after lifting")
	}
	if b.typed() != "1" || !b.input.IsKeyJustPressed(ebiten.KeyDigit1) {
		t.Errorf("a fresh landing typed %q and pressed %v", b.typed(), b.input.AppendJustPressedKeys(nil))
	}
}

func TestAValidReplacementShowsFirstLayerAtOnceAndReleasesOnUpdate(t *testing.T) {
	b := compactBoard(t)
	switchLayer(t, b, "123")
	b.touch(t, 4, "1")
	b.Update()

	if err := b.SetLayout(osk.Layout{}); err == nil {
		t.Fatal("accepted a layout without layers")
	}
	b.Update()
	if !b.input.IsKeyPressed(ebiten.KeyDigit1) || b.key(t, "1").State != osk.StatePressed {
		t.Fatal("an invalid replacement dropped the held key")
	}

	if err := b.SetLayout(stockBoard(t, osk.EsES)); err != nil {
		t.Fatal(err)
	}
	if b.key(t, "Ñ").State != osk.StateIdle || !b.Shown() {
		t.Fatal("a replacement did not immediately show idle Spanish letters")
	}
	b.Update()
	checkRelease(t, b, ebiten.KeyDigit1)
	if !b.PointerTaken(4) {
		t.Error("the layout replacement gave up the pointer before lift")
	}

	b.reland(t, 4, "Ñ")
	b.Update()
	if b.typed() != "Ñ" || !b.input.IsKeyJustPressed(ebiten.KeySemicolon) {
		t.Errorf("a fresh landing typed %q and pressed %v", b.typed(), b.input.AppendJustPressedKeys(nil))
	}
}

func checkRelease(t *testing.T, b board, position ebiten.Key) {
	t.Helper()
	want := []osk.Event{{Kind: osk.KeyUp, Key: position}}
	if got := b.AppendEvents(nil); !slices.Equal(got, want) {
		t.Errorf("after switching, got events %v, want %v", got, want)
	}
	if !b.input.IsKeyJustReleased(position) || b.input.IsKeyPressed(position) || b.typed() != "" {
		t.Errorf("after switching, released %v, pressed %v, typed %q",
			b.input.AppendJustReleasedKeys(nil), b.input.AppendPressedKeys(nil), b.typed())
	}
	b.Update()
	if len(b.AppendEvents(nil)) != 0 || b.input.IsKeyJustReleased(position) || b.typed() != "" {
		t.Error("a pointer that stayed down reported input on the following tick")
	}
}

func TestNamedLayerTargetsAreValidatedAndCopied(t *testing.T) {
	layout := osk.Layout{
		Layers: []osk.Layer{
			{Name: "start", Rows: [][]osk.Key{{{Do: osk.SwitchLayer("next"), Label: "Next"}}}},
			{Name: "next", Rows: [][]osk.Key{{{Do: osk.Type("x")}}}},
		},
		Positions: osk.Positions{'x': ebiten.KeyX},
	}
	b := newBoard(t, layout)
	layout.Layers[1].Name = "start"
	if _, err := osk.New(layout); err == nil {
		t.Error("duplicate layer names accepted")
	}
	layout.Layers[1].Name = ""
	if _, err := osk.New(layout); err == nil {
		t.Error("unknown layer target accepted")
	}
	layout.Layers[1].Rows[0][0].Do = osk.Type("z")
	layout.Positions['x'] = ebiten.KeyZ

	switchLayer(t, b, "Next")
	b.touch(t, 2, "x")
	b.Update()
	if b.typed() != "x" || !b.input.IsKeyPressed(ebiten.KeyX) {
		t.Errorf("the copied layout typed %q or pressed %v", b.typed(), b.input.AppendPressedKeys(nil))
	}
}
