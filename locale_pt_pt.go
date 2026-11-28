package osk

import "github.com/hajimehoshi/ebiten/v2"

// PtPT selects the Portuguese layout for Portugal.
const PtPT Locale = "pt-PT"

var portuguese = fullLocale{
	letterRows:   [3]string{"qwertyuiop", "asdfghjklç", "zxcvbnm"},
	symbolRows:   [3]string{"1234567890", ",-_'?!@:/\"", ";+=\\()«»<"},
	variants:     map[rune]string{'a': "áàâã", 'e': "éê", 'i': "í", 'o': "óôõ", 'u': "ú"},
	space:        "Espaço",
	shift:        "Maiús",
	erase:        "Apagar",
	actionLabels: [...]string{"Introduzir", "Seguinte", "Concluir", "Ir", "Pesquisar", "Enviar"},
}

func compactPtPT() Layout {
	return portuguese.compact(withLettersAndDigits(portugueseBase))
}

func fullPtPT(kind Kind, action FieldAction) Layout {
	return portuguese.layout(withLettersAndDigits(portugueseBase), kind, action)
}

// portugueseBase was checked against xkeyboard-config's pt(basic).
var portugueseBase = Positions{
	' ': ebiten.KeySpace, '\\': ebiten.KeyBackquote,
	'\'': ebiten.KeyMinus, '«': ebiten.KeyEqual, '+': ebiten.KeyBracketLeft,
	'ç': ebiten.KeySemicolon, 'Ç': ebiten.KeySemicolon, 'º': ebiten.KeyQuote,
	',': ebiten.KeyComma, '.': ebiten.KeyPeriod, '-': ebiten.KeySlash,
	'<': ebiten.KeyIntlBackslash,
}
