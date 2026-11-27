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
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
	text.SetColor(ink)
	text.SetAlign(etxt.Center)

	// The physical keyboard types what it likes: a character the font lacks is drawn as the
	// font's mark for a missing one, where etxt would otherwise panic.
	text.Glyph().SetMissHandler(etxt.OnMissNotdef)

	return &game{keyboard: keyboard, painter: paint.New(theme), text: text}, nil
}

// Update reads the keyboard exactly as a game reads a physical one, with osk in the place of
// ebiten and inpututil.
func (g *game) Update() error {
	if err := g.chooseLayout(); err != nil {
		return err
	}
	g.keyboard.Update()

	if osk.IsKeyJustPressed(ebiten.KeyBackspace) {
		g.line = g.line[:max(len(g.line)-1, 0)]
	}

	g.line = osk.AppendInputChars(g.line)

	return nil
}

func (g *game) chooseLayout() error {
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyF1):
		return g.showCompact(osk.EnUS)
	case inpututil.IsKeyJustPressed(ebiten.KeyF2):
		return g.showCompact(osk.EsES)
	case inpututil.IsKeyJustPressed(ebiten.KeyF3):
		return g.keyboard.SetLayout(pinPad())
	default:
		return nil
	}
}

func (g *game) showCompact(locale osk.Locale) error {
	layout, err := osk.Stock(locale, osk.Compact)
	if err != nil {
		return err
	}
	return g.keyboard.SetLayout(layout)
}

// pinPad builds a layout from the same blocks as the stock boards.
func pinPad() osk.Layout {
	positions := make(osk.Positions, 10)
	rows := make([][]osk.Key, 4)
	for i, digits := range []string{"123", "456", "789", "0"} {
		for _, digit := range digits {
			positions[digit] = ebiten.KeyDigit0 + ebiten.Key(digit-'0')
			rows[i] = append(rows[i], osk.Key{Do: osk.Type(string(digit))})
		}
	}
	rows[3] = append(rows[3], osk.Key{Do: osk.Erase(), Label: "Delete", Weight: 2})

	return osk.Layout{Positions: positions, Layers: []osk.Layer{{Rows: rows}}}
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(background)
	g.text.SetSize(48)
	g.text.Draw(screen, string(g.line)+"_", width/2, boardTop/2)
	g.text.SetSize(20)
	g.text.Draw(screen, "F1  en-US     F2  es-ES     F3  PIN pad", width/2, boardTop-28)
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
