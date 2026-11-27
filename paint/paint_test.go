package paint_test

import (
	"image"
	"image/color"
	"slices"
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	"github.com/hajimehoshi/ebiten/v2"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/bstkhq/go-ebiten-osk/paint"
)

// The screen of the test game, and the part of it the keyboard takes: four rows of 60 pixels,
// ten keys of 64 across.
const (
	screenW, screenH = 640, 360
	boardTop         = 120
)

// typist is a game with an on-screen keyboard that reads it as any game reads a keyboard,
// through the input copy, and keeps what it read.
type typist struct {
	keyboard *osk.Keyboard
	painter  *paint.Painter

	text     []rune
	pressed  []ebiten.Key
	released []ebiten.Key
}

func newTypist() ebiten.Game {
	layout, err := osk.Stock(osk.EnUS, osk.Compact)
	if err != nil {
		panic(err)
	}

	keyboard, err := osk.New(layout)
	if err != nil {
		panic(err)
	}

	keyboard.SetRect(image.Rect(0, boardTop, screenW, screenH))

	return &typist{keyboard: keyboard, painter: paint.New(paint.Dark())}
}

func (g *typist) Update() error {
	g.keyboard.Update()

	if osk.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.text) > 0 {
		g.text = g.text[:len(g.text)-1]
	}

	g.text = osk.AppendInputChars(g.text)
	g.pressed = osk.AppendJustPressedKeys(g.pressed)
	g.released = osk.AppendJustReleasedKeys(g.released)

	return nil
}

func (g *typist) Draw(screen *ebiten.Image) {
	g.painter.Draw(screen, g.keyboard)
}

func (g *typist) Layout(int, int) (int, int) {
	return screenW, screenH
}

func TestMain(m *testing.M) {
	ebitenmcp.RunTests(m, newTypist)
}

// hands are the test's hands on the running game.
type hands struct {
	*ebitenmcp.Driver
	t *testing.T
}

func sit(t *testing.T) hands {
	t.Helper()

	return hands{Driver: ebitenmcp.T(t), t: t}
}

// read gives what the game holds to look, between two ticks.
func (h hands) read(look func(*typist)) {
	h.t.Helper()

	h.WithGame(func(game ebiten.Game) { look(game.(*typist)) })
}

// middle is the middle of the key that shows face.
func (h hands) middle(face string) image.Point {
	h.t.Helper()

	var at image.Point

	found := false

	h.read(func(g *typist) {
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

	h.read(func(g *typist) { text = string(g.text) })

	return text
}

func TestAGameReadsClicksOnTheKeysThroughTheInputCopy(t *testing.T) {
	h := sit(t)

	h.click("H")
	h.click("I")

	if got := h.text(); got != "HI" {
		t.Errorf("clicking H and I typed %q", got)
	}

	h.read(func(g *typist) {
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

	h.read(func(g *typist) {
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

	h.read(func(g *typist) {
		if !slices.Contains(g.pressed, ebiten.KeyBackspace) {
			t.Errorf("the game saw %v pressed, want the physical backspace among them", g.pressed)
		}
	})
}

func TestTheDarkCompactBoard(t *testing.T) {
	h := sit(t)

	h.Tick(1)
	h.Golden("compact_en_us_dark.png")
}

func TestAPressedKeyLooksPressed(t *testing.T) {
	h := sit(t)

	theme := paint.Dark()
	g, y := h.middle("G"), h.middle("Y")

	// Left of the label, at mid height: the key's face in both states.
	probe := image.Pt(g.X-20, g.Y)

	h.Move(float64(g.X), float64(g.Y))

	if got := colourAt(h.Screenshot(), probe); got != theme.Key {
		t.Fatalf("an idle key's face is %v, want the theme's %v", got, theme.Key)
	}

	h.Runtime().Injector().MouseDown(ebiten.MouseButtonLeft)
	h.Tick(2)

	if got := colourAt(h.Screenshot(), probe); got != theme.Pressed {
		t.Errorf("a key under the mouse is %v, want the theme's pressed %v", got, theme.Pressed)
	}

	if got := colourAt(h.Screenshot(), image.Pt(y.X-20, y.Y)); got != theme.Key {
		t.Errorf("a key nothing touches is %v while another is pressed, want %v", got, theme.Key)
	}

	h.Runtime().Injector().MouseUp(ebiten.MouseButtonLeft)
	h.Tick(2)

	if got := colourAt(h.Screenshot(), probe); got != theme.Key {
		t.Errorf("a released key's face is %v, want %v again", got, theme.Key)
	}
}

func TestAHiddenKeyboardIsNotDrawn(t *testing.T) {
	h := sit(t)

	g := h.middle("G")

	h.Tick(1)

	if got := colourAt(h.Screenshot(), image.Pt(g.X-20, g.Y)); got != paint.Dark().Key {
		t.Fatalf("before it is hidden, the key is %v, want %v", got, paint.Dark().Key)
	}

	h.read(func(g *typist) { g.keyboard.Hide() })
	h.Tick(2)

	if got := colourAt(h.Screenshot(), image.Pt(g.X-20, g.Y)); got.R != 0 || got.G != 0 || got.B != 0 {
		t.Errorf("where a hidden keyboard was, the screen is %v, want nothing drawn", got)
	}
}

func colourAt(screen *image.RGBA, p image.Point) color.RGBA {
	return screen.RGBAAt(p.X, p.Y)
}
