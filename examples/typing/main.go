// Command typing shows the on-screen keyboard typing a line of text.
//
// The game reads its keyboard through the input copy, so the keys on the screen and the keys of
// the physical keyboard both type:
//
//	go run github.com/bstkhq/go-ebiten-osk/examples/typing@latest
package main

import (
	"image"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tinne26/etxt"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/bstkhq/go-ebiten-osk/paint"
)

const (
	width, height = 960, 540

	// boardTop is where the keyboard starts: it takes the lower part of the screen.
	boardTop = 240
)

var (
	background = color.RGBA{0x24, 0x27, 0x2d, 0xff}
	ink        = color.RGBA{0xf1, 0xf2, 0xf4, 0xff}
)

// game is a line of text and the keyboard that types it.
type game struct {
	keyboard *osk.Keyboard
	painter  *paint.Painter
	text     *etxt.Renderer
	line     []rune
}

func newGame() (*game, error) {
	layout, err := osk.Stock(osk.EnUS, osk.Compact)
	if err != nil {
		return nil, err
	}

	keyboard, err := osk.New(layout)
	if err != nil {
		return nil, err
	}

	keyboard.SetRect(image.Rect(0, boardTop, width, height))

	theme := paint.Dark()

	text := etxt.NewRenderer()
	text.SetFont(theme.Font)
	text.SetSize(48)
	text.SetColor(ink)
	text.SetAlign(etxt.Center)

	return &game{keyboard: keyboard, painter: paint.New(theme), text: text}, nil
}

// Update reads the keyboard exactly as a game reads a physical one, with osk in the place of
// ebiten and inpututil.
func (g *game) Update() error {
	g.keyboard.Update()

	if osk.IsKeyJustPressed(ebiten.KeyBackspace) {
		g.line = g.line[:max(len(g.line)-1, 0)]
	}

	g.line = osk.AppendInputChars(g.line)

	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(background)
	g.text.Draw(screen, string(g.line)+"_", width/2, boardTop/2)
	g.painter.Draw(screen, g.keyboard)
}

func (g *game) Layout(int, int) (int, int) {
	return width, height
}

func run() error {
	g, err := newGame()
	if err != nil {
		return err
	}

	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowTitle("go-ebiten-osk: typing")

	return ebiten.RunGame(g)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
