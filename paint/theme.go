// Package paint draws an on-screen keyboard from its state and a theme, with go-shapes and
// etxt.
//
// A [Painter] reads a keyboard's keys, where each is and how it looks now, and draws them as
// its [Theme] says. Nobody makes a theme from scratch: take a stock one and change what you
// want.
//
//	theme := paint.Dark()
//	theme.Font = myFont
//	theme.Key = color.RGBA{0x20, 0x40, 0x60, 0xff}
//	painter := paint.New(theme)
//
//	painter.Draw(screen, keyboard) // in the game's Draw
//
// A game with a look that values cannot reach draws its keyboard itself, from
// [osk.Keyboard.AppendKeys], and needs nothing of this package.
package paint

import (
	"image/color"
	"sync"

	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/sfnt"
)

// A Theme is the values that give a keyboard its look. Every field is used as it stands:
// start from a stock theme, such as [Dark], and change the ones you want.
type Theme struct {
	// Panel is the colour behind the keys.
	Panel color.RGBA
	// Key is the colour of a key's face, and Pressed its colour under a finger.
	Key     color.RGBA
	Pressed color.RGBA
	// Lip is the colour of a key's lower edge, which the face covers while it is pressed.
	Lip color.RGBA
	// Label is the colour of what a key shows.
	Label color.RGBA

	// Font draws the labels.
	Font *sfnt.Font

	// The sizes are fractions of a row's height, so a keyboard looks the same at any size.

	// CharacterSize is the size of a label of one character, and WordSize of a longer one.
	// A label too wide for its key is drawn smaller, so that it fits.
	CharacterSize float64
	WordSize      float64
	// Gap is the space between two keys.
	Gap float64
	// Radius is the rounding of a key's corners.
	Radius float64
	// LipHeight is how much of a key's lower edge shows under its face.
	LipHeight float64
}

// Dark is the stock dark theme: light labels on charcoal keys, in Go Bold.
func Dark() Theme {
	return Theme{
		Panel:   color.RGBA{0x17, 0x19, 0x1d, 0xff},
		Key:     color.RGBA{0x34, 0x38, 0x40, 0xff},
		Pressed: color.RGBA{0x55, 0x5b, 0x67, 0xff},
		Lip:     color.RGBA{0x0c, 0x0d, 0x10, 0xff},
		Label:   color.RGBA{0xf1, 0xf2, 0xf4, 0xff},

		Font: goBold(),

		CharacterSize: 0.44,
		WordSize:      0.30,
		Gap:           0.10,
		Radius:        0.12,
		LipHeight:     0.07,
	}
}

// goBold is Go Bold, parsed once: every stock theme shares it.
var goBold = sync.OnceValue(parseGoBold)

func parseGoBold() *sfnt.Font {
	font, err := sfnt.Parse(gobold.TTF)
	if err != nil {
		panic("paint: Go Bold, which ships with x/image, does not parse: " + err.Error())
	}

	return font
}
