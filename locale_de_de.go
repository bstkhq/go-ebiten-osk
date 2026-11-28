package osk

import "github.com/hajimehoshi/ebiten/v2"

// DeDE is the stock locale for German in Germany.
const DeDE Locale = "de-DE"

var deDE = fullLocale{
	letterRows:   [3]string{"qwertzuiopü", "asdfghjklöä", "yxcvbnmß"},
	symbolRows:   [3]string{"1234567890", ",-_'?!@:/&", "+#<>;()\"="},
	space:        "Leerzeichen",
	shift:        "Umschalt",
	erase:        "Löschen",
	actionLabels: [...]string{"Eingabe", "Weiter", "Fertig", "Los", "Suchen", "Senden"},
}

func compactDeDE() Layout { return deDE.compact(positionsDeDE()) }

func fullDeDE(kind Kind, action FieldAction) Layout {
	return deDE.layout(positionsDeDE(), kind, action)
}

// positionsDeDE was checked against xkeyboard-config's de(basic). The
// circumflex, acute and grave are dead keys on that board.
func positionsDeDE() Positions {
	positions := withLettersAndDigits(Positions{
		' ': ebiten.KeySpace,
		'ü': ebiten.KeyBracketLeft, 'Ü': ebiten.KeyBracketLeft, 'ß': ebiten.KeyMinus,
		'ö': ebiten.KeySemicolon, 'Ö': ebiten.KeySemicolon, 'ä': ebiten.KeyQuote, 'Ä': ebiten.KeyQuote,
		'+': ebiten.KeyBracketRight, '#': ebiten.KeyBackslash, '<': ebiten.KeyIntlBackslash,
		',': ebiten.KeyComma, '.': ebiten.KeyPeriod, '-': ebiten.KeySlash,
	})
	positions['y'], positions['Y'] = ebiten.KeyZ, ebiten.KeyZ
	positions['z'], positions['Z'] = ebiten.KeyY, ebiten.KeyY
	return positions
}
