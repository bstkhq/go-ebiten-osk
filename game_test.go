package osk_test

import (
	"image"
	"slices"
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	"github.com/hajimehoshi/ebiten/v2"

	osk "github.com/bstkhq/go-ebiten-osk"
)

// The tests of this package run beside one game loop, go-ebiten-mcp's: a test that takes its
// driver gets a fresh reader and moves it tick by tick. The other tests update keyboards of
// their own by hand and never touch the game.
func TestMain(m *testing.M) {
	ebitenmcp.RunTests(m, newReader)
}

// reader is a game with an on-screen keyboard that reads it as any game reads a keyboard: with
// the package's own functions, where it had ebiten's and inpututil's. It keeps what it read.
type reader struct {
	keyboard *osk.Keyboard

	text     []rune
	pressed  []ebiten.Key
	released []ebiten.Key

	// What it saw of H: whether it is down, the keys down with it, how long it was held at
	// most, and how many times it went up.
	hDown    bool
	down     []ebiten.Key
	hLongest int
	hUps     int
}

func newReader() ebiten.Game {
	layout, err := osk.Stock(osk.EnUS, osk.Compact)
	if err != nil {
		panic(err)
	}

	keyboard, err := osk.New(layout)
	if err != nil {
		panic(err)
	}

	keyboard.SetRect(image.Rect(0, 120, 640, 360))

	return &reader{keyboard: keyboard}
}

func (g *reader) Update() error {
	if g.keyboard != nil {
		g.keyboard.Update()
	}

	if osk.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.text) > 0 {
		g.text = g.text[:len(g.text)-1]
	}

	g.text = osk.AppendInputChars(g.text)
	g.pressed = osk.AppendJustPressedKeys(g.pressed)
	g.released = osk.AppendJustReleasedKeys(g.released)

	g.hDown = osk.IsKeyPressed(ebiten.KeyH)
	g.down = osk.AppendPressedKeys(g.down[:0])
	g.hLongest = max(g.hLongest, osk.KeyPressDuration(ebiten.KeyH))

	if osk.IsKeyJustReleased(ebiten.KeyH) {
		g.hUps++
	}

	return nil
}

func (g *reader) Draw(*ebiten.Image) {}

func (g *reader) Layout(int, int) (int, int) {
	return 640, 360
}

// hands are a test's hands on the running game.
type hands struct {
	*ebitenmcp.Driver
	t *testing.T
}

func sit(t *testing.T) hands {
	t.Helper()

	return hands{Driver: ebitenmcp.T(t), t: t}
}

// read gives what the game holds to look, between two ticks.
func (h hands) read(look func(*reader)) {
	h.t.Helper()

	h.WithGame(func(game ebiten.Game) { look(game.(*reader)) })
}

// middle is the middle of the key that shows face.
func (h hands) middle(face string) image.Point {
	h.t.Helper()

	var at image.Point

	found := false

	h.read(func(g *reader) {
		for _, key := range g.keyboard.AppendKeys(nil) {
			if key.Face() == face {
				at, found = key.Rect.Min.Add(key.Rect.Size().Div(2)), true
			}
		}
	})

	if !found {
		h.t.Fatalf("the keyboard shows no key %q", face)
	}

	return at
}

func (h hands) click(face string) {
	h.t.Helper()

	at := h.middle(face)
	h.Click(float64(at.X), float64(at.Y))
}

func (h hands) text() string {
	h.t.Helper()

	var text string

	h.read(func(g *reader) { text = string(g.text) })

	return text
}

func TestAGameReadsClicksOnTheKeysWithThePackagesFunctions(t *testing.T) {
	h := sit(t)

	h.click("H")
	h.click("I")

	if got := h.text(); got != "HI" {
		t.Errorf("clicking H and I typed %q", got)
	}

	h.read(func(g *reader) {
		want := []ebiten.Key{ebiten.KeyH, ebiten.KeyI}

		if !slices.Equal(g.pressed, want) || !slices.Equal(g.released, want) {
			t.Errorf("the game saw %v pressed and %v released, want %v of each", g.pressed, g.released, want)
		}
	})

	h.click("Space")
	h.click("Delete")
	h.click("Delete")

	if got := h.text(); got != "H" {
		t.Errorf("a space and two erases after HI left %q, want H", got)
	}
}

func TestAGameSeesAKeyHeldOnTheScreenForAsLongAsItIsHeld(t *testing.T) {
	h := sit(t)

	at := h.middle("H")
	h.Move(float64(at.X), float64(at.Y))

	mouse := h.Runtime().Injector()
	mouse.MouseDown(ebiten.MouseButtonLeft)
	h.Tick(6)

	h.read(func(g *reader) {
		if !g.hDown || !slices.Equal(g.down, []ebiten.Key{ebiten.KeyH}) {
			t.Errorf("with the mouse held on H the game saw H down: %t, and these keys down: %v", g.hDown, g.down)
		}

		if g.hLongest < 4 || g.hUps != 0 {
			t.Errorf("after six ticks on H the game saw it held for %d ticks and released %d times", g.hLongest, g.hUps)
		}
	})

	if got := h.text(); got != "H" {
		t.Errorf("a key held for six ticks typed %q, want one H", got)
	}

	mouse.MouseUp(ebiten.MouseButtonLeft)
	h.Tick(3)

	h.read(func(g *reader) {
		if g.hDown || g.hUps != 1 || len(g.down) != 0 {
			t.Errorf("after the mouse let go the game saw H down: %t, released %d times, keys down: %v", g.hDown, g.hUps, g.down)
		}
	})
}

func TestAGameReadsTouchesOnTheKeys(t *testing.T) {
	h := sit(t)

	o, k := h.middle("O"), h.middle("K")

	fingers := h.Runtime().Injector()
	fingers.SetTouches([]ebitenmcp.Touch{{ID: 3, X: o.X, Y: o.Y}})
	h.Tick(2)
	fingers.SetTouches([]ebitenmcp.Touch{{ID: 3, X: o.X, Y: o.Y}, {ID: 4, X: k.X, Y: k.Y}})
	h.Tick(2)
	fingers.SetTouches(nil)
	h.Tick(2)

	if got := h.text(); got != "OK" {
		t.Errorf("a finger on O and then another on K typed %q", got)
	}

	h.read(func(g *reader) {
		if !slices.Contains(g.released, ebiten.KeyO) || !slices.Contains(g.released, ebiten.KeyK) {
			t.Errorf("lifting both fingers released %v, want O and K", g.released)
		}
	})
}

func TestTheSameGameStillReadsThePhysicalKeyboard(t *testing.T) {
	h := sit(t)

	h.click("H")
	h.Type("ey")
	h.Tap(ebiten.KeyBackspace)

	if got := h.text(); got != "He" {
		t.Errorf("H on the screen, then e, y and backspace on the keyboard, left %q, want He", got)
	}

	h.read(func(g *reader) {
		if !slices.Contains(g.pressed, ebiten.KeyBackspace) || !slices.Contains(g.released, ebiten.KeyBackspace) {
			t.Errorf("the game saw %v pressed and %v released, want the physical backspace in both", g.pressed, g.released)
		}
	})
}

func TestAGameThatDropsItsOnlyKeyboardSeesItsKeysGoUpOnce(t *testing.T) {
	h := sit(t)

	at := h.middle("H")
	h.Move(float64(at.X), float64(at.Y))
	h.Runtime().Injector().MouseDown(ebiten.MouseButtonLeft)
	h.Tick(2)

	// The text field closes while the mouse still holds H: the game drops its keyboard and
	// goes on reading the input, with no other keyboard in its place.
	h.read(func(g *reader) { g.keyboard = nil })
	h.Tick(6)

	h.read(func(g *reader) {
		if g.hDown || len(g.down) != 0 {
			t.Errorf("six ticks after its keyboard was dropped, the game still sees H down: %t, keys down: %v", g.hDown, g.down)
		}

		if g.hUps != 1 || string(g.text) != "H" {
			t.Errorf("the dropped keyboard's H went up %d times and left the text %q, want once and one H", g.hUps, string(g.text))
		}
	})

	h.Runtime().Injector().MouseUp(ebiten.MouseButtonLeft)
	h.Tick(1)

	// The physical H afterwards is a press like any other.
	h.Tap(ebiten.KeyH)

	h.read(func(g *reader) {
		want := []ebiten.Key{ebiten.KeyH, ebiten.KeyH}

		if !slices.Equal(g.pressed, want) || !slices.Equal(g.released, want) {
			t.Errorf("H on the dropped keyboard and then on the physical one: the game saw %v pressed and %v released, want %v of each",
				g.pressed, g.released, want)
		}
	})
}
