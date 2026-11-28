package osk

import "github.com/hajimehoshi/ebiten/v2"

// ItIT is the stock locale for Italian in Italy.
const ItIT Locale = "it-IT"

var itIT = fullLocale{
	letterRows:   [3]string{"qwertyuiop", "asdfghjkl", "zxcvbnm"},
	symbolRows:   [3]string{"1234567890", ",-_'?!@:/&", "\\+<>;()\"£"},
	variants:     map[rune]string{'a': "à", 'e': "èé", 'i': "ì", 'o': "ò", 'u': "ù"},
	space:        "Spazio",
	shift:        "Maiusc",
	erase:        "Cancella",
	actionLabels: [...]string{"Invio", "Avanti", "Fine", "Vai", "Cerca", "Invia"},
}

func compactItIT() Layout { return itIT.compact(positionsItIT()) }

func fullItIT(kind Kind, action FieldAction) Layout {
	return itIT.layout(positionsItIT(), kind, action)
}

// positionsItIT was checked against xkeyboard-config's it(basic). È keeps the
// position of è when capitalized. The shifted letter é has no base position.
func positionsItIT() Positions {
	return withLettersAndDigits(Positions{
		' ': ebiten.KeySpace,
		'è': ebiten.KeyBracketLeft, 'È': ebiten.KeyBracketLeft,
		'à': ebiten.KeyQuote, 'À': ebiten.KeyQuote, 'ì': ebiten.KeyEqual, 'Ì': ebiten.KeyEqual,
		'ò': ebiten.KeySemicolon, 'Ò': ebiten.KeySemicolon, 'ù': ebiten.KeyBackslash, 'Ù': ebiten.KeyBackslash,
		'\\': ebiten.KeyBackquote, '\'': ebiten.KeyMinus, '+': ebiten.KeyBracketRight,
		'<': ebiten.KeyIntlBackslash, ',': ebiten.KeyComma, '.': ebiten.KeyPeriod, '-': ebiten.KeySlash,
	})
}
