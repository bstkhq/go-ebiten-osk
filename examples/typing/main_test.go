package main

import (
	"image"
	"image/color"
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bstkhq/go-ebiten-osk/paint"
)

func TestMain(m *testing.M) {
	ebitenmcp.RunTests(m, func() ebiten.Game {
		g, err := newGame(false)
		if err != nil {
			panic(err)
		}

		return g
	})
}

func TestTheExampleTypesItsLineFromTheScreenAndFromTheKeyboard(t *testing.T) {
	d := ebitenmcp.T(t)

	if inked(d.Screenshot()) < 20 {
		t.Fatal("before anything is typed, the example does not draw its caret")
	}

	caret := inked(d.Screenshot())

	// Q is the first key of the board: the top left corner of its rectangle.
	d.Click(48, boardTop+30)
	d.Type("a")
	d.Click(48, boardTop+30)
	d.Tap(ebiten.KeyBackspace)

	d.WithGame(func(g ebiten.Game) {
		if got := string(g.(*game).line); got != "Qa" {
			t.Errorf("Q on the screen, a on the keyboard, Q again and a backspace left the line %q, want Qa", got)
		}
	})

	if got := inked(d.Screenshot()); got <= caret {
		t.Errorf("the typed line is drawn with %d pixels of ink, no more than the %d of the caret alone", got, caret)
	}
}

func TestTheExampleDrawsACharacterItsFontLacks(t *testing.T) {
	d := ebitenmcp.T(t)

	// A Hebrew keyboard types a letter Go Bold has no glyph for.
	d.Type("א")

	// The driver fails the test if the game panics while it draws these frames.
	d.Tick(3)

	d.WithGame(func(g ebiten.Game) {
		if got := string(g.(*game).line); got != "א" {
			t.Errorf("a letter the font lacks left the line %q, want the letter kept", got)
		}
	})

	if inked(d.Screenshot()) == 0 {
		t.Error("with a letter the font lacks in its line, the example draws nothing")
	}
}

func TestTheExampleSwitchesThemesAndTypesWithItsOwnPainter(t *testing.T) {
	d := ebitenmcp.T(t)
	at := image.Pt(20, boardTop+25)
	for _, want := range []color.RGBA{paint.Light().Palette.Key, customKey, paint.Dark().Palette.Key} {
		d.Tap(ebiten.KeyF2)
		if got := d.Screenshot().RGBAAt(at.X, at.Y); got != want {
			t.Errorf("after F2 the key colour is %v, want %v", got, want)
		}
		if want == customKey {
			d.Click(48, boardTop+30)
		}
	}
	d.WithGame(func(g ebiten.Game) {
		if got := string(g.(*game).line); got != "Q" {
			t.Errorf("the custom painter's board typed %q, want Q", got)
		}
	})
}

// inked counts the pixels of the text's colour above the keyboard.
func inked(screen *image.RGBA) int {
	count := 0

	for y := range boardTop {
		for x := range width {
			if screen.RGBAAt(x, y) == ink {
				count++
			}
		}
	}

	return count
}
