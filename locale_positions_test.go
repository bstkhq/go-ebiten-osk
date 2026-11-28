package osk_test

import (
	"slices"
	"testing"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestStockLocalesListsRegisteredBoards(t *testing.T) {
	locales := osk.StockLocales()
	for _, locale := range []osk.Locale{"en-US", "es-ES", "en-GB", "de-DE", "it-IT"} {
		if !slices.Contains(locales, locale) {
			t.Errorf("StockLocales() is missing %s", locale)
		}
	}
	if !slices.IsSorted(locales) {
		t.Errorf("StockLocales() is not sorted by tag: %v", locales)
	}
	for _, locale := range locales {
		if _, err := osk.Stock(locale, osk.Compact); err != nil {
			t.Errorf("StockLocales() includes %s with no stock layout: %v", locale, err)
		}
	}
}

var britishPositions = map[rune]ebiten.Key{
	'.': ebiten.KeyPeriod, ',': ebiten.KeyComma, '-': ebiten.KeyMinus,
	'\'': ebiten.KeyQuote, '/': ebiten.KeySlash, '`': ebiten.KeyBackquote,
	'=': ebiten.KeyEqual, '[': ebiten.KeyBracketLeft, ']': ebiten.KeyBracketRight,
	'\\': ebiten.KeyIntlBackslash, ';': ebiten.KeySemicolon, '#': ebiten.KeyBackslash,
}

var germanPositions = map[rune]ebiten.Key{
	'y': ebiten.KeyZ, 'Y': ebiten.KeyZ, 'z': ebiten.KeyY, 'Z': ebiten.KeyY,
	'ü': ebiten.KeyBracketLeft, 'Ü': ebiten.KeyBracketLeft, 'ß': ebiten.KeyMinus,
	'ö': ebiten.KeySemicolon, 'Ö': ebiten.KeySemicolon, 'ä': ebiten.KeyQuote, 'Ä': ebiten.KeyQuote,
	'+': ebiten.KeyBracketRight, '#': ebiten.KeyBackslash, '<': ebiten.KeyIntlBackslash,
	',': ebiten.KeyComma, '.': ebiten.KeyPeriod, '-': ebiten.KeySlash,
}

var italianPositions = map[rune]ebiten.Key{
	'è': ebiten.KeyBracketLeft, 'È': ebiten.KeyBracketLeft,
	'à': ebiten.KeyQuote, 'À': ebiten.KeyQuote, 'ì': ebiten.KeyEqual, 'Ì': ebiten.KeyEqual,
	'ò': ebiten.KeySemicolon, 'Ò': ebiten.KeySemicolon, 'ù': ebiten.KeyBackslash, 'Ù': ebiten.KeyBackslash,
	'\\': ebiten.KeyBackquote, '\'': ebiten.KeyMinus, '+': ebiten.KeyBracketRight,
	'<': ebiten.KeyIntlBackslash, ',': ebiten.KeyComma, '.': ebiten.KeyPeriod, '-': ebiten.KeySlash,
}

func TestFullGermanCapitalsKeepTheirPhysicalPositions(t *testing.T) {
	b := fullBoard(t, osk.Locale("de-DE"))
	for range 2 {
		b.tap(t, 1, "Umschalt")
		b.Update()
	}
	for _, face := range []string{"Z", "Y", "Ä", "Ö", "Ü", "ß"} {
		checkTyping(t, b, face, []string{"Leerzeichen", "Löschen"}, germanPositions)
	}
}
