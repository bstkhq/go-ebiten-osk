# go-ebiten-osk

An on-screen keyboard for [Ebitengine](https://ebitengine.org) games. Draw it in
your game and read touch, mouse, and physical keyboard input through the same
keyboard-reading functions.

## Features

- Compact and full layouts in eight locales, with text, email, and number kinds.
- Long-press accent selection in supported locales.
- Multiple touches, repeating erase, and configurable field actions.
- Dark and light themes, custom layouts and painters, and animated show and hide.

## Example

From a checkout, save this as `example/main.go` and run `go run ./example`. Tap
or click the keys, or type on your physical keyboard.

```go
package main

import (
	"image"
	"log"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/bstkhq/go-ebiten-osk/paint"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const width, height = 640, 360

type game struct {
	keyboard *osk.Keyboard
	painter  *paint.Painter
	text     []rune
}

func (g *game) Update() error {
	g.keyboard.Update()
	if osk.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.text) > 0 {
		g.text = g.text[:len(g.text)-1]
	}
	g.text = osk.AppendInputChars(g.text)
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, string(g.text))
	g.painter.Draw(screen, g.keyboard)
}

func (*game) Layout(int, int) (int, int) { return width, height }

func main() {
	layout, err := osk.Stock(osk.EnUS, osk.Compact)
	if err != nil {
		log.Fatal(err)
	}
	keyboard, err := osk.New(layout)
	if err != nil {
		log.Fatal(err)
	}
	keyboard.SetRect(image.Rect(0, 140, width, height))
	ebiten.SetWindowSize(width, height)
	if err := ebiten.RunGame(&game{keyboard: keyboard, painter: paint.New(paint.Dark())}); err != nil {
		log.Fatal(err)
	}
}
```

## Demo

From the repository root, run `go run ./examples/typing`. Select Name, Email, or
Number to try the different layouts.

## Documentation

- [Input, pointer ownership, and the optional Ebitengine hook](docs/input.md)
- [Stock locales, custom layouts, and field actions](docs/layouts.md)
- [Themes, custom painters, and show/hide animation](docs/appearance.md)
- [Demo controls and development checks](docs/development.md)
- [Glossary](GLOSSARY.md) and [input design decision](docs/adr/0001-a-game-reads-the-input-copy.md)

## License

[MIT](LICENSE).
