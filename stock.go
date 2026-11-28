package osk

import (
	"fmt"
	"maps"

	"github.com/hajimehoshi/ebiten/v2"
)

// Stock is the layout the library ships for a locale in a form.
func Stock(locale Locale, form Form) (Layout, error) {
	return StockWithAction(locale, form, ActionEnter)
}

// StockWithAction looks up a stock text layout with the requested field action.
// Compact text layouts have no action key.
func StockWithAction(locale Locale, form Form, action FieldAction) (Layout, error) {
	return StockWithKind(locale, form, KindText, action)
}

// StockWithKind looks up a stock layout for the field. Full email layouts show
// @ on the letters layer. Compact email uses the compact text layout. Number
// uses the same localized pad in either form. Pass the layout to [New] or
// [Keyboard.SetLayout] to select the kind for a keyboard.
func StockWithKind(locale Locale, form Form, kind Kind, action FieldAction) (Layout, error) {
	if !action.valid() {
		return Layout{}, fmt.Errorf("osk: unknown field action %v", action)
	}
	if kind > KindNumber {
		return Layout{}, fmt.Errorf("osk: unknown kind %d", kind)
	}
	build, ok := stock[locale]
	if !ok || (form != Compact && form != Full) {
		return Layout{}, fmt.Errorf("osk: no stock %v layout for the locale %q", form, locale)
	}

	if form == Compact && kind != KindNumber {
		return build.compact(), nil
	}
	return build.full(kind, action), nil
}

var stock = map[Locale]struct {
	compact func() Layout
	full    func(Kind, FieldAction) Layout
}{
	FrFR: {compactFrFR, fullFrFR},
	PtPT: {compactPtPT, fullPtPT},
	CaES: {compactCaES, fullCaES},
	EnUS: {compactEnUS, fullEnUS},
	EsES: {compactEsES, fullEsES},
}

const (
	letters = "letters"
	digits  = "digits"
	accents = "accents"

	stockRowWidth = 10
)

func compactEnUS() Layout {
	space := Key{Do: Type(" "), Label: "Space", Weight: 5}
	erase := Key{Do: Erase(), Label: "Delete", Weight: 3}

	return Layout{
		Positions: positionsEnUS(),
		Layers: []Layer{
			compactLayer(letters, [3]string{"QWERTYUIOP", "ASDFGHJKL", "ZXCVBNM"},
				layerSwitch(digits, "123"), space, erase),
			compactLayer(digits, [3]string{"1234567890", ".,-_'?!@:/", "`=[]\\;()&\""},
				layerSwitch(letters, "ABC"), space, erase),
		},
	}
}

func compactEsES() Layout {
	space := Key{Do: Type(" "), Label: "Espacio", Weight: 4}
	erase := Key{Do: Erase(), Label: "Borrar", Weight: 2}
	toLetters := layerSwitch(letters, "ABC")
	toDigits := layerSwitch(digits, "123")
	toAccents := layerSwitch(accents, "ÁÉÍ")

	return Layout{
		Positions: positionsEsES(),
		Layers: []Layer{
			compactLayer(letters, [3]string{"QWERTYUIOP", "ASDFGHJKLÑ", "ZXCVBNM"},
				toDigits, toAccents, space, erase),
			compactLayer(digits, [3]string{"1234567890", ".,-_'¿?¡!@", "`+´:;()/º<"},
				toLetters, toAccents, space, erase),
			compactLayer(accents, [3]string{"ÁÉÍ", "ÓÚ", "ÜÇ"},
				toLetters, toDigits, space, erase),
		},
	}
}

// compactLayer puts the controls under three rows of characters.
func compactLayer(name string, rows [3]string, controls ...Key) Layer {
	return Layer{Name: name, Rows: [][]Key{
		characters(rows[0], stockRowWidth), characters(rows[1], stockRowWidth),
		characters(rows[2], stockRowWidth), controls,
	}}
}

func layerSwitch(name, label string) Key {
	return Key{Do: SwitchLayer(name), Label: label, Weight: 2}
}

// characters shares the available width equally when a stock row has more
// characters than can fit at full width.
func characters(row string, width float64) []Key {
	keys := make([]Key, 0, len(row))

	for _, character := range row {
		keys = append(keys, Key{Do: Type(string(character))})
	}

	if float64(len(keys)) > width {
		for i := range keys {
			keys[i].Weight = width / float64(len(keys))
		}
	}

	return keys
}

// positionsEnUS is the base level of a US physical board, and its capitals: with caps lock a
// capital letter presses its letter's key and nothing else.
func positionsEnUS() Positions {
	return withLettersAndDigits(Positions{
		' ': ebiten.KeySpace,
		'`': ebiten.KeyBackquote, '-': ebiten.KeyMinus, '=': ebiten.KeyEqual,
		'[': ebiten.KeyBracketLeft, ']': ebiten.KeyBracketRight, '\\': ebiten.KeyBackslash,
		';': ebiten.KeySemicolon, '\'': ebiten.KeyQuote,
		',': ebiten.KeyComma, '.': ebiten.KeyPeriod, '/': ebiten.KeySlash,
	})
}

// positionsEsES follows the Spanish physical positions checked against
// xkeyboard-config's es(basic), including the positions of the standalone grave
// and acute marks. Composed letters type text without pressing those positions.
func positionsEsES() Positions {
	return withLettersAndDigits(Positions{
		' ': ebiten.KeySpace,
		'ñ': ebiten.KeySemicolon, 'Ñ': ebiten.KeySemicolon,
		'ç': ebiten.KeyBackslash, 'Ç': ebiten.KeyBackslash,
		'º': ebiten.KeyBackquote, '<': ebiten.KeyIntlBackslash,
		'\'': ebiten.KeyMinus, '¡': ebiten.KeyEqual,
		'`': ebiten.KeyBracketLeft, '+': ebiten.KeyBracketRight,
		'´': ebiten.KeyQuote,
		',': ebiten.KeyComma, '.': ebiten.KeyPeriod, '-': ebiten.KeySlash,
	})
}

func withLettersAndDigits(base Positions) Positions {
	positions := make(Positions, len(base)+52+10)
	maps.Copy(positions, base)
	for letter := 'a'; letter <= 'z'; letter++ {
		key := ebiten.KeyA + ebiten.Key(letter-'a')
		positions[letter] = key
		positions[letter-'a'+'A'] = key
	}

	for digit := '0'; digit <= '9'; digit++ {
		positions[digit] = ebiten.KeyDigit0 + ebiten.Key(digit-'0')
	}

	return positions
}
