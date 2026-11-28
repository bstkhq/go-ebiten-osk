package osk

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// Stock is the layout the library ships for a locale in a form.
func Stock(locale Locale, form Form) (Layout, error) {
	build, ok := stock[stockKey{locale, form}]
	if !ok {
		return Layout{}, fmt.Errorf("osk: no stock %v layout for the locale %q", form, locale)
	}

	return build(), nil
}

type stockKey struct {
	locale Locale
	form   Form
}

var stock = map[stockKey]func() Layout{
	{EnUS, Compact}: compactEnUS,
	{EsES, Compact}: compactEsES,
}

const (
	letters = "letters"
	digits  = "digits"
	accents = "accents"
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
		characters(rows[0]), characters(rows[1]), characters(rows[2]), controls,
	}}
}

func layerSwitch(name, label string) Key {
	return Key{Do: SwitchLayer(name), Label: label, Weight: 2}
}

// characters is a row of keys that each type one of the characters, in their order.
func characters(row string) []Key {
	keys := make([]Key, 0, len(row))

	for _, character := range row {
		keys = append(keys, Key{Do: Type(string(character))})
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

func withLettersAndDigits(positions Positions) Positions {
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
