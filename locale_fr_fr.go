package osk

import "github.com/hajimehoshi/ebiten/v2"

// FrFR selects the traditional French AZERTY layout.
const FrFR Locale = "fr-FR"

var french = fullLocale{
	letterRows:   [3]string{"azertyuiop", "qsdfghjklm", "wxcvbn"},
	symbolRows:   [3]string{"1234567890", ",-_'?!@:&\"", ";()=+*/$<"},
	accentRows:   [3]string{"éèêë", "àâç", "ùûîïôœ"},
	accentLabel:  "ÉÈÀ",
	space:        "Espace",
	shift:        "Maj",
	erase:        "Effacer",
	actionLabels: [...]string{"Entrée", "Suivant", "Terminé", "Aller", "Rechercher", "Envoyer"},
}

func compactFrFR() Layout {
	return french.compact(positionsFrFR())
}

func fullFrFR(kind Kind, action FieldAction) Layout {
	return french.layout(positionsFrFR(), kind, action)
}

// positionsFrFR was checked against xkeyboard-config's fr(basic).
// The stock digit keys keep their physical digit positions on AZERTY.
func positionsFrFR() Positions {
	positions := withLettersAndDigits(Positions{
		' ': ebiten.KeySpace, '²': ebiten.KeyBackquote,
		'&': ebiten.KeyDigit1, 'é': ebiten.KeyDigit2, 'É': ebiten.KeyDigit2,
		'"': ebiten.KeyDigit3, '\'': ebiten.KeyDigit4, '(': ebiten.KeyDigit5,
		'-': ebiten.KeyDigit6, 'è': ebiten.KeyDigit7, 'È': ebiten.KeyDigit7,
		'_': ebiten.KeyDigit8, 'ç': ebiten.KeyDigit9, 'Ç': ebiten.KeyDigit9,
		'à': ebiten.KeyDigit0, 'À': ebiten.KeyDigit0, ')': ebiten.KeyMinus, '=': ebiten.KeyEqual,
		'$': ebiten.KeyBracketRight, 'ù': ebiten.KeyQuote, 'Ù': ebiten.KeyQuote,
		'*': ebiten.KeyBackslash, '<': ebiten.KeyIntlBackslash,
		',': ebiten.KeyM, ';': ebiten.KeyComma, ':': ebiten.KeyPeriod, '!': ebiten.KeySlash,
	})
	positions['a'], positions['A'] = ebiten.KeyQ, ebiten.KeyQ
	positions['z'], positions['Z'] = ebiten.KeyW, ebiten.KeyW
	positions['q'], positions['Q'] = ebiten.KeyA, ebiten.KeyA
	positions['w'], positions['W'] = ebiten.KeyZ, ebiten.KeyZ
	positions['m'], positions['M'] = ebiten.KeySemicolon, ebiten.KeySemicolon
	return positions
}
