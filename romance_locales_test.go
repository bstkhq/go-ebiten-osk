package osk_test

import (
	"image"
	"slices"
	"strings"
	"testing"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
)

type localeCase struct {
	locale                      osk.Locale
	letters, symbols, accents   []string
	space, shift, erase, accent string
	actions                     [osk.ActionSend + 1]string
	positions                   map[rune]ebiten.Key
}

var localeCases = []localeCase{
	{
		locale: "fr-FR", letters: []string{"azertyuiop", "qsdfghjklm", "wxcvbn"},
		symbols: []string{"1234567890", ",-_'?!@:&\"", ";()=+*/$<"},
		accents: []string{"éèêë", "àâç", "ùûîïôœ"},
		space:   "Espace", shift: "Maj", erase: "Effacer", accent: "ÉÈÀ",
		actions: [...]string{"Entrée", "Suivant", "Terminé", "Aller", "Rechercher", "Envoyer"},
		positions: map[rune]ebiten.Key{
			'a': ebiten.KeyQ, 'A': ebiten.KeyQ, 'z': ebiten.KeyW, 'Z': ebiten.KeyW,
			'q': ebiten.KeyA, 'Q': ebiten.KeyA, 'w': ebiten.KeyZ, 'W': ebiten.KeyZ,
			'm': ebiten.KeySemicolon, 'M': ebiten.KeySemicolon, ',': ebiten.KeyM,
			';': ebiten.KeyComma, ':': ebiten.KeyPeriod, '!': ebiten.KeySlash,
			'&': ebiten.KeyDigit1, 'é': ebiten.KeyDigit2, 'É': ebiten.KeyDigit2,
			'"': ebiten.KeyDigit3, '\'': ebiten.KeyDigit4, '(': ebiten.KeyDigit5,
			'-': ebiten.KeyDigit6, 'è': ebiten.KeyDigit7, 'È': ebiten.KeyDigit7,
			'_': ebiten.KeyDigit8, 'ç': ebiten.KeyDigit9, 'Ç': ebiten.KeyDigit9,
			'à': ebiten.KeyDigit0, 'À': ebiten.KeyDigit0, ')': ebiten.KeyMinus,
			'=': ebiten.KeyEqual, '$': ebiten.KeyBracketRight, 'ù': ebiten.KeyQuote,
			'Ù': ebiten.KeyQuote, '*': ebiten.KeyBackslash, '<': ebiten.KeyIntlBackslash,
		},
	},
	{
		locale: "pt-PT", letters: []string{"qwertyuiop", "asdfghjklç", "zxcvbnm"},
		symbols: []string{"1234567890", ",-_'?!@:/\"", ";+=\\()«»<"},
		accents: []string{"áàâã", "éêí", "óôõú"},
		space:   "Espaço", shift: "Maiús", erase: "Apagar", accent: "ÁÃÕ",
		actions: [...]string{"Introduzir", "Seguinte", "Concluir", "Ir", "Pesquisar", "Enviar"},
		positions: map[rune]ebiten.Key{
			'ç': ebiten.KeySemicolon, 'Ç': ebiten.KeySemicolon,
			'\\': ebiten.KeyBackquote, '\'': ebiten.KeyMinus, '«': ebiten.KeyEqual,
			'+': ebiten.KeyBracketLeft, ',': ebiten.KeyComma, '.': ebiten.KeyPeriod,
			'-': ebiten.KeySlash, '<': ebiten.KeyIntlBackslash,
		},
	},
	{
		locale: "ca-ES", letters: []string{"qwertyuiop", "asdfghjklñç", "zxcvbnm"},
		symbols: []string{"1234567890", ",-_'¿?¡!@+", "`´:;()/º<"},
		accents: []string{"àèé", "íïòó", "úü·"},
		space:   "Espai", shift: "Majús", erase: "Esborra", accent: "ÀÈÉ",
		actions: [...]string{"Retorn", "Següent", "Fet", "Vés", "Cerca", "Envia"},
		positions: map[rune]ebiten.Key{
			'ñ': ebiten.KeySemicolon, 'Ñ': ebiten.KeySemicolon,
			'ç': ebiten.KeyBackslash, 'Ç': ebiten.KeyBackslash, 'º': ebiten.KeyBackquote,
			'<': ebiten.KeyIntlBackslash, '\'': ebiten.KeyMinus, '¡': ebiten.KeyEqual,
			'`': ebiten.KeyBracketLeft, '´': ebiten.KeyQuote, '+': ebiten.KeyBracketRight,
			',': ebiten.KeyComma, '.': ebiten.KeyPeriod, '-': ebiten.KeySlash,
		},
	},
}

func TestAdditionalCompactLocaleLayers(t *testing.T) {
	for _, locale := range localeCases {
		for _, kind := range []osk.Kind{osk.KindText, osk.KindEmail} {
			name := string(locale.locale) + []string{"/text", "/email"}[kind]
			t.Run(name, func(t *testing.T) {
				b := newKindBoard(t, locale.locale, osk.Compact, kind, osk.ActionEnter)
				b.SetRect(image.Rect(7, 3, 1004, 398))
				home := b.key(t, "123").Rect
				controls := []string{locale.space, locale.erase}
				letters := strings.ToUpper(strings.Join(locale.letters, ""))
				checkLayer(t, b, letters, controls, []string{"123", locale.accent}, locale.positions)
				checkCompactLetterOrder(t, b, locale.letters)

				switchLayer(t, b, "123")
				checkLayer(t, b, strings.Join(locale.symbols, "")+".", controls, []string{"ABC", locale.accent}, locale.positions)
				checkFullReturn(t, b, home, letters[:1])

				switchLayer(t, b, locale.accent)
				checkLayer(t, b, strings.ToUpper(strings.Join(locale.accents, "")), controls, []string{"ABC", "123"}, locale.positions)
				checkFullReturn(t, b, home, letters[:1])
			})
		}
	}
}

func checkCompactLetterOrder(t *testing.T, b board, rows []string) {
	t.Helper()
	keys := b.AppendKeys(nil)
	for i, row := range rows {
		top := b.Rect().Min.Y + i*b.Rect().Dy()/4
		for _, face := range strings.Split(strings.ToUpper(row), "") {
			if len(keys) == 0 {
				t.Fatalf("row %d is missing %q", i, face)
			}
			key := keys[0]
			keys = keys[1:]
			if key.Face() != face || key.Rect.Min.Y != top {
				t.Fatalf("row %d has %q at y=%d, want %q at y=%d", i, key.Face(), key.Rect.Min.Y, face, top)
			}
		}
	}
}

func TestAdditionalFullLocaleLayers(t *testing.T) {
	for _, locale := range localeCases {
		for _, kind := range []osk.Kind{osk.KindText, osk.KindEmail} {
			name := string(locale.locale) + []string{"/text", "/email"}[kind]
			t.Run(name, func(t *testing.T) {
				b := newKindBoard(t, locale.locale, osk.Full, kind, osk.ActionEnter)
				b.SetRect(image.Rect(7, 3, 1004, 398))
				home := b.key(t, "123").Rect
				bottom := []string{"123", locale.accent, locale.space, ".", locale.actions[osk.ActionEnter]}
				letters := slices.Clone(bottom)
				if kind == osk.KindEmail {
					letters = slices.Insert(letters, 2, "@")
				}
				checkFullLayer(t, b, fullRows(locale.letters, letters, locale.shift, locale.erase), locale.space, locale.erase, locale.positions)

				switchLayer(t, b, "123")
				bottom[0] = "ABC"
				checkFullLayer(t, b, fullRows(locale.symbols, bottom, "", locale.erase), locale.space, locale.erase, locale.positions)
				checkFullReturn(t, b, home, locale.letters[0][:1])

				switchLayer(t, b, locale.accent)
				bottom[1] = "123"
				checkFullLayer(t, b, fullRows(locale.accents, bottom, locale.shift, locale.erase), locale.space, locale.erase, locale.positions)
				for range 2 {
					switchLayer(t, b, locale.shift)
				}
				capitals := []string{
					strings.ToUpper(locale.accents[0]), strings.ToUpper(locale.accents[1]), strings.ToUpper(locale.accents[2]),
				}
				checkFullLayer(t, b, fullRows(capitals, bottom, locale.shift, locale.erase), locale.space, locale.erase, locale.positions)
				checkFullReturn(t, b, home, strings.ToUpper(locale.letters[0][:1]))
				checkFullLayer(t, b, fullRows([]string{
					strings.ToUpper(locale.letters[0]), strings.ToUpper(locale.letters[1]), strings.ToUpper(locale.letters[2]),
				}, letters, locale.shift, locale.erase), locale.space, locale.erase, locale.positions)
			})
		}
	}
}

func TestAdditionalLocaleNumberPadsAndActions(t *testing.T) {
	for _, locale := range localeCases {
		t.Run(string(locale.locale), func(t *testing.T) {
			for _, form := range []osk.Form{osk.Compact, osk.Full} {
				b := newKindBoard(t, locale.locale, form, osk.KindNumber, osk.ActionDone)
				done := locale.actions[osk.ActionDone]
				rows := [][]string{{"1", "2", "3"}, {"4", "5", "6"}, {"7", "8", "9"}, {locale.erase, "0", done}}
				checkFullLayer(t, b, rows, locale.space, locale.erase, locale.positions)
				checkLocalizedAction(t, b, osk.ActionDone, done)
			}
			for action, label := range locale.actions {
				b := newKindBoard(t, locale.locale, osk.Full, osk.KindText, osk.FieldAction(action))
				checkLocalizedAction(t, b, osk.FieldAction(action), label)
			}
		})
	}
}

func checkLocalizedAction(t *testing.T, b board, action osk.FieldAction, label string) {
	t.Helper()
	if got := b.key(t, label).Role(); got != osk.RoleAction {
		t.Errorf("%q has role %v, want action", label, got)
	}
	b.tap(t, 1, label)
	b.Update()
	want := osk.Event{Kind: osk.ActionPressed, FieldAction: action}
	if !b.input.IsKeyJustPressed(ebiten.KeyEnter) || !slices.Contains(b.AppendEvents(nil), want) || b.typed() != "" {
		t.Errorf("tapping %q reported %v and typed %q, want Enter and %v", label, b.AppendEvents(nil), b.typed(), want)
	}
}
