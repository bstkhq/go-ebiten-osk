package osk

import "github.com/hajimehoshi/ebiten/v2"

// EnGB is the stock locale for British English.
const EnGB Locale = "en-GB"

var enGB = fullLocale{
	letterRows:   [3]string{"qwertyuiop", "asdfghjkl", "zxcvbnm"},
	symbolRows:   [3]string{"1234567890", ",-_'?!@:/&£", "`=[]\\;()\"#"},
	space:        "Space",
	shift:        "Shift",
	erase:        "Backspace",
	actionLabels: [...]string{"Return", "Next", "Done", "Go", "Search", "Send"},
}

func compactEnGB() Layout { return enGB.compact(positionsEnGB()) }

func fullEnGB(kind Kind, action FieldAction) Layout {
	return enGB.layout(positionsEnGB(), kind, action)
}

// positionsEnGB was checked against xkeyboard-config's gb(basic). The ISO
// key beside left Shift types a backslash. The key beside Return types #.
func positionsEnGB() Positions {
	positions := positionsEnUS()
	positions['\\'] = ebiten.KeyIntlBackslash
	positions['#'] = ebiten.KeyBackslash
	return positions
}
