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
}

// compactEnUS is the letters of a US board in capitals, with space and erase under them.
func compactEnUS() Layout {
	return Layout{
		Positions: positionsEnUS(),
		Layers: []Layer{{Rows: [][]Key{
			characters("QWERTYUIOP"),
			characters("ASDFGHJKL"),
			characters("ZXCVBNM"),
			{
				{Do: Type(" "), Label: "Space", Weight: 7},
				{Do: Erase(), Label: "Delete", Weight: 3},
			},
		}}},
	}
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
	positions := Positions{
		' ': ebiten.KeySpace,
		'`': ebiten.KeyBackquote, '-': ebiten.KeyMinus, '=': ebiten.KeyEqual,
		'[': ebiten.KeyBracketLeft, ']': ebiten.KeyBracketRight, '\\': ebiten.KeyBackslash,
		';': ebiten.KeySemicolon, '\'': ebiten.KeyQuote,
		',': ebiten.KeyComma, '.': ebiten.KeyPeriod, '/': ebiten.KeySlash,
	}

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
