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
		g, err := newGame()
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
		if got := string(g.(*game).values[0]); got != "Qa" {
			t.Errorf("Q on the screen, a on the keyboard, Q again and a backspace left the line %q, want Qa", got)
		}
	})

	if got := inked(d.Screenshot()); got <= caret {
		t.Errorf("the typed line is drawn with %d pixels of ink, no more than the %d of the caret alone", got, caret)
	}
}

func TestTheExampleOpensTheKeysForEachField(t *testing.T) {
	d := ebitenmcp.T(t)
	d.Click(48, boardTop+37) // Q in Name.
	d.Click(width/2, boardTop/2)
	d.Click(48, boardTop+37) // q in Email, with the full layout active.
	d.Click(240, height-37)  // @ beside space.
	d.Click(width*5/6, boardTop/2)
	d.Click(width/6, boardTop+37) // 1 on the number pad.
	d.Click(width/2, boardTop+37) // 2 beside it.
	d.Click(width/6, height-37)   // erase on the number pad.
	d.WithGame(func(g ebiten.Game) {
		for index, want := range []string{"Q", "q@", "1"} {
			if got := string(g.(*game).values[index]); got != want {
				t.Errorf("%s contains %q, want %q", fields[index].label, got, want)
			}
		}
	})
}

func TestTheExampleDrawsACharacterItsFontLacks(t *testing.T) {
	d := ebitenmcp.T(t)

	// A Hebrew keyboard types a letter Go Bold has no glyph for.
	d.Type("א")

	// The driver fails the test if the game panics while it draws these frames.
	d.Tick(3)

	d.WithGame(func(g ebiten.Game) {
		if got := string(g.(*game).values[0]); got != "א" {
			t.Errorf("a letter the font lacks left the line %q, want the letter kept", got)
		}
	})

	if inked(d.Screenshot()) == 0 {
		t.Error("with a letter the font lacks in its line, the example draws nothing")
	}
}

func TestTheExampleAcceptsATouchOnAFieldAndNextOnTheKeyboard(t *testing.T) {
	d := ebitenmcp.T(t)
	injector := d.Runtime().Injector()
	injector.SetTouches([]ebitenmcp.Touch{{ID: 1, X: width / 2, Y: boardTop / 2}})
	d.Tick(2)
	injector.SetTouches(nil)
	d.Tick(2)

	d.Click(48, boardTop+37)      // q on the full email keyboard.
	d.Click(width-48, height-37)  // Next brings up the number pad.
	d.Click(width/6, boardTop+37) // 1 on the number pad.
	d.WithGame(func(g ebiten.Game) {
		for index, want := range []string{"", "q", "1"} {
			if got := string(g.(*game).values[index]); got != want {
				t.Errorf("%s contains %q, want %q", fields[index].label, got, want)
			}
		}
	})
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
		if got := string(g.(*game).values[0]); got != "Q" {
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
