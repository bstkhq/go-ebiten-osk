package main

import (
	"image"
	"image/color"
	"testing"
	"time"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	"github.com/hajimehoshi/ebiten/v2"

	osk "github.com/bstkhq/go-ebiten-osk"
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
	d.Click(288, height-37)  // @ beside space.
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
	injector.SetTouches([]ebitenmcp.Touch{
		{ID: 1, X: 24, Y: 200},                 // Outside any field.
		{ID: 2, X: width / 2, Y: boardTop / 2}, // Email.
	})
	d.Tick(2)
	injector.SetTouches(nil)
	d.Tick(2)

	d.Click(48, boardTop+37)      // q on the full email keyboard.
	d.Click(width-48, height-37)  // Next brings up the number pad.
	d.Click(width/6, boardTop+37) // 1 on the number pad.
	d.Click(width*5/6, height-37) // Done leaves the number field selected.
	d.Click(width/2, boardTop+37) // 2 on the number pad.
	d.WithGame(func(g ebiten.Game) {
		for index, want := range []string{"", "q", "12"} {
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

func TestTheExampleShowsAndHidesWithAndWithoutAnimation(t *testing.T) {
	d := ebitenmcp.T(t)
	at := image.Pt(20, boardTop+25)
	requirePartialBoard(t, firstFrameAfterF7(t, d))
	d.Tick(15)
	if got := d.Screenshot().RGBAAt(at.X, at.Y); got != background {
		t.Errorf("after hiding, the board pixel is %v, want the background", got)
	}

	d.Tap(ebiten.KeyF8)
	d.Tap(ebiten.KeyF7)
	if got := d.Screenshot().RGBAAt(at.X, at.Y); got != paint.Dark().Palette.Key {
		t.Errorf("after showing with animation off, the key colour is %v", got)
	}
	d.Tap(ebiten.KeyF7)
	if got := d.Screenshot().RGBAAt(at.X, at.Y); got != background {
		t.Errorf("after hiding with animation off, the board pixel is %v", got)
	}
	d.Tap(ebiten.KeyF8)
	requirePartialBoard(t, firstFrameAfterF7(t, d))
	d.Tick(15)
	if got := d.Screenshot().RGBAAt(at.X, at.Y); got != paint.Dark().Palette.Key {
		t.Errorf("after showing with animation back on, the key colour is %v", got)
	}
}

func firstFrameAfterF7(t *testing.T, d *ebitenmcp.Driver) *image.RGBA {
	t.Helper()
	runtime := d.Runtime()
	inject := runtime.Injector()
	// Inject F7 and request the step in one engine tick. Step pauses the game
	// again after that update, leaving the first transition frame to capture.
	var tick int64
	d.WithGame(func(ebiten.Game) {
		tick = runtime.Tick()
		inject.KeyDown(ebiten.KeyF7)
		runtime.Step(1)
	})
	defer func() {
		d.WithGame(func(ebiten.Game) {
			inject.KeyUp(ebiten.KeyF7)
			runtime.Resume()
		})
	}()
	deadline := time.Now().Add(5 * time.Second)
	for runtime.Tick() == tick && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if runtime.Tick() != tick+1 {
		t.Fatal("the typing example did not advance one tick on F7")
	}
	return d.Screenshot()
}

func requirePartialBoard(t *testing.T, frame *image.RGBA) {
	t.Helper()
	if got := frame.RGBAAt(2, boardTop); got != background {
		t.Errorf("at the start of motion the top edge is %v, want the background", got)
	}
	if got := frame.RGBAAt(2, height-10); got == background {
		t.Error("at the start of motion the bottom of the board should be visible")
	}
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
func TestTheExampleAdvancesAndSubmitsWithBothKeyboards(t *testing.T) {
	checkFormActions(t, ebitenmcp.T(t))
}

func TestTheExampleSubmitsEachDistinctActionInOneTick(t *testing.T) {
	d := ebitenmcp.T(t)
	d.Click(width*5/6, boardTop/2) // Focus Number.
	layout, err := osk.StockWithKind(osk.EnUS, osk.Compact, osk.KindNumber, osk.ActionDone)
	if err != nil {
		t.Fatal(err)
	}
	taps := pendingTaps{
		{ID: 1, X: width * 5 / 6, Y: height - 37, Landed: true, Lifted: true},
		{ID: 2, X: width * 5 / 6, Y: height - 37, Landed: true, Lifted: true},
	}
	d.WithGame(func(current ebiten.Game) {
		g := current.(*game)
		var keyboard *osk.Keyboard
		keyboard, err = osk.New(layout, osk.WithPointers(&taps), osk.WithActionHandler(g.queueAction))
		if err == nil {
			keyboard.SetRect(g.keyboard.Rect())
			g.keyboard = keyboard
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	d.Tick(2)
	checkFormState(t, d, 2, 2)
}

type pendingTaps []osk.Pointer

func (taps *pendingTaps) AppendPointers(pointers []osk.Pointer) []osk.Pointer {
	pointers = append(pointers, (*taps)...)
	*taps = nil
	return pointers
}

func checkFormActions(t *testing.T, d *ebitenmcp.Driver) {
	t.Helper()
	d.Tap(ebiten.KeyF3)          // Show Next on the full layout for Name.
	d.Click(width-48, height-37) // Next must focus Email once.
	checkFormState(t, d, 1, 0)
	d.Tap(ebiten.KeyEnter) // Physical Enter focuses Number.
	checkFormState(t, d, 2, 0)
	d.Click(width*5/6, height-37) // Done submits once.
	checkFormState(t, d, 2, 1)
	d.Tap(ebiten.KeyEnter) // Physical Enter submits again.
	checkFormState(t, d, 2, 2)
}

func checkFormState(t *testing.T, d *ebitenmcp.Driver, field, submissions int) {
	t.Helper()
	d.WithGame(func(current ebiten.Game) {
		g := current.(*game)
		if g.active != field || g.submissions != submissions {
			t.Errorf("active field = %s and submissions = %d, want %s and %d",
				fields[g.active].label, g.submissions, fields[field].label, submissions)
		}
	})
}
