package paint_test

import (
	"image"
	"image/color"
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

// board is a game that is a keyboard and the painter that draws it.
type board struct {
	keyboard *osk.Keyboard
	painter  *paint.Painter
}

func newBoard() ebiten.Game {
	layout, err := osk.Stock(osk.EnUS, osk.Compact)
	if err != nil {
		panic(err)
	}

	keyboard, err := osk.New(layout)
	if err != nil {
		panic(err)
	}

	keyboard.SetRect(image.Rect(0, boardTop, screenW, screenH))

	return &board{keyboard: keyboard, painter: paint.New(paint.Dark())}
}

func (g *board) Update() error {
	g.keyboard.Update()

	return nil
}

func (g *board) Draw(screen *ebiten.Image) {
	g.painter.Draw(screen, g.keyboard)
}

func (g *board) Layout(int, int) (int, int) {
	return screenW, screenH
}

func TestMain(m *testing.M) {
	ebitenmcp.RunTests(m, newBoard)
}

// face is a point on the face of the key that shows it, in both states: left of the label, at
// mid height.
func face(t *testing.T, d *ebitenmcp.Driver, shows string) image.Point {
	t.Helper()

	var at image.Point

	found := false

	d.WithGame(func(game ebiten.Game) {
		for _, key := range game.(*board).keyboard.AppendKeys(nil) {
			if key.Face() == shows {
				at, found = key.Rect.Min.Add(key.Rect.Size().Div(2)), true
			}
		}
	})

	if !found {
		t.Fatalf("the keyboard shows no key %q", shows)
	}

	return at.Sub(image.Pt(20, 0))
}

func colourAt(screen *image.RGBA, p image.Point) color.RGBA {
	return screen.RGBAAt(p.X, p.Y)
}

func TestTheDarkCompactBoard(t *testing.T) {
	d := ebitenmcp.T(t)

	d.Tick(1)
	d.Golden("compact_en_us_dark.png")
}

func TestAPressedKeyLooksPressed(t *testing.T) {
	d := ebitenmcp.T(t)

	theme := paint.Dark()
	g, y := face(t, d, "G"), face(t, d, "Y")

	d.Move(float64(g.X), float64(g.Y))

	if got := colourAt(d.Screenshot(), g); got != theme.Key {
		t.Fatalf("an idle key's face is %v, want the theme's %v", got, theme.Key)
	}

	d.Runtime().Injector().MouseDown(ebiten.MouseButtonLeft)
	d.Tick(2)

	if got := colourAt(d.Screenshot(), g); got != theme.Pressed {
		t.Errorf("a key under the mouse is %v, want the theme's pressed %v", got, theme.Pressed)
	}

	if got := colourAt(d.Screenshot(), y); got != theme.Key {
		t.Errorf("a key nothing touches is %v while another is pressed, want %v", got, theme.Key)
	}

	d.Runtime().Injector().MouseUp(ebiten.MouseButtonLeft)
	d.Tick(2)

	if got := colourAt(d.Screenshot(), g); got != theme.Key {
		t.Errorf("a released key's face is %v, want %v again", got, theme.Key)
	}
}

func TestAPainterDrawsWhatItsThemeSays(t *testing.T) {
	d := ebitenmcp.T(t)

	theme := paint.Dark()
	theme.Panel = color.RGBA{0x10, 0x40, 0x70, 0xff}
	theme.Key = color.RGBA{0x70, 0x20, 0x20, 0xff}

	g := face(t, d, "G")

	d.WithGame(func(game ebiten.Game) { game.(*board).painter = paint.New(theme) })
	d.Tick(1)

	if got := colourAt(d.Screenshot(), g); got != theme.Key {
		t.Errorf("with the key's colour changed, a key's face is %v, want %v", got, theme.Key)
	}

	// Beside the centred second row there is no key: only the panel.
	if got := colourAt(d.Screenshot(), image.Pt(8, boardTop+90)); got != theme.Panel {
		t.Errorf("with the panel's colour changed, the panel is %v, want %v", got, theme.Panel)
	}
}

func TestALabelTooWideForItsKeyIsDrawnSmallerNotCut(t *testing.T) {
	d := ebitenmcp.T(t)

	wide := osk.Layout{Layers: []osk.Layer{{Rows: [][]osk.Key{{
		{Do: osk.Type("w"), Label: "WWWWWWWWWWWW"},
		{Do: osk.Type("x"), Weight: 9},
	}}}}}

	d.WithGame(func(game ebiten.Game) {
		keyboard, err := osk.New(wide)
		if err != nil {
			t.Fatal(err)
		}

		keyboard.SetRect(image.Rect(0, boardTop, screenW, screenH))
		game.(*board).keyboard = keyboard
	})
	d.Tick(1)

	// The first key is 64 pixels wide. Its label is the only light thing on it, and small
	// letters are never quite the label's colour: none of that ink may reach the key's
	// edges or the next key.
	screen := d.Screenshot()
	faceRed := paint.Dark().Key.R

	inked := 0

	for y := boardTop; y < screenH; y++ {
		for x := range 64 {
			if colourAt(screen, image.Pt(x, y)).R < faceRed+0x40 {
				continue
			}

			inked++

			if x < 6 || x > 58 {
				t.Fatalf("the label's ink reaches x = %d, outside its key's face", x)
			}
		}
	}

	if inked == 0 {
		t.Error("the label was not drawn at all")
	}
}

func TestALabelTheFontLacksIsDrawnWithoutStoppingTheGame(t *testing.T) {
	d := ebitenmcp.T(t)

	// Go Bold has no Hebrew: a layout of a game's own may ask for a letter the theme's font
	// cannot draw.
	foreign := osk.Layout{Layers: []osk.Layer{{Rows: [][]osk.Key{{
		{Do: osk.Type("א")},
		{Do: osk.Type("x")},
	}}}}}

	d.WithGame(func(game ebiten.Game) {
		keyboard, err := osk.New(foreign)
		if err != nil {
			t.Fatal(err)
		}

		keyboard.SetRect(image.Rect(0, boardTop, screenW, screenH))
		game.(*board).keyboard = keyboard
	})

	// The driver fails the test if the game panics while it draws these frames.
	d.Tick(3)

	if got := colourAt(d.Screenshot(), image.Pt(20, boardTop+120)); got != paint.Dark().Key {
		t.Errorf("the key with a label the font lacks has a face of %v, want the theme's %v", got, paint.Dark().Key)
	}

	if got := colourAt(d.Screenshot(), image.Pt(screenW-20, boardTop+120)); got != paint.Dark().Key {
		t.Errorf("the key beside it has a face of %v, want the theme's %v", got, paint.Dark().Key)
	}
}

func TestAThemeMadeFromScratchStillDrawsABoard(t *testing.T) {
	d := ebitenmcp.T(t)

	// Against the advice to start from a stock theme: two colours, and no font or size.
	bare := paint.Theme{Key: color.RGBA{0x20, 0x60, 0x20, 0xff}, Label: color.RGBA{0xff, 0xff, 0xff, 0xff}}
	g := face(t, d, "G")

	d.WithGame(func(game ebiten.Game) { game.(*board).painter = paint.New(bare) })
	d.Tick(3)

	screen := d.Screenshot()

	if got := colourAt(screen, g); got != bare.Key {
		t.Errorf("a theme with only a key colour draws a key's face %v, want %v", got, bare.Key)
	}

	if lit := labelInk(screen, bare.Label); lit != 0 {
		t.Errorf("a theme that gives labels no size drew %d pixels of label", lit)
	}

	// With a size and still no font, the labels are in Go Bold.
	bare.CharacterSize = 0.4

	d.WithGame(func(game ebiten.Game) { game.(*board).painter = paint.New(bare) })
	d.Tick(2)

	if lit := labelInk(d.Screenshot(), bare.Label); lit == 0 {
		t.Error("a theme with a label size and no font drew no label")
	}
}

// labelInk counts the pixels of exactly the label's colour on the board.
func labelInk(screen *image.RGBA, label color.RGBA) int {
	count := 0

	for y := boardTop; y < screenH; y++ {
		for x := range screenW {
			if colourAt(screen, image.Pt(x, y)) == label {
				count++
			}
		}
	}

	return count
}

func TestAHiddenKeyboardIsNotDrawn(t *testing.T) {
	d := ebitenmcp.T(t)

	g := face(t, d, "G")

	d.Tick(1)

	if got := colourAt(d.Screenshot(), g); got != paint.Dark().Key {
		t.Fatalf("before it is hidden, the key is %v, want %v", got, paint.Dark().Key)
	}

	d.WithGame(func(game ebiten.Game) { game.(*board).keyboard.Hide() })
	d.Tick(2)

	if got := colourAt(d.Screenshot(), g); got.R != 0 || got.G != 0 || got.B != 0 {
		t.Errorf("where a hidden keyboard was, the screen is %v, want nothing drawn", got)
	}
}
