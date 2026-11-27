# go-ebiten-osk

[![Checks](https://github.com/bstkhq/go-ebiten-osk/actions/workflows/checks.yml/badge.svg?branch=main)](https://github.com/bstkhq/go-ebiten-osk/actions/workflows/checks.yml)

An on-screen keyboard for [Ebitengine](https://ebitengine.org) that a game reads like a physical one.

A game on a touchscreen with no system keyboard (a kiosk, a cabinet) has no way to let people type. This library draws a keyboard in the game, and lets the game read it with the functions it already uses: the library has a copy of Ebitengine's keyboard-reading functions, with the same names and signatures, that answers for the physical keyboard and the on-screen one together. Moving a game to the on-screen keyboard is a change of import.

```go
import (
	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/bstkhq/go-ebiten-osk/paint"
)

layout, _ := osk.Stock(osk.EnUS, osk.Compact)
keyboard, _ := osk.New(layout)
keyboard.SetRect(image.Rect(0, 240, 960, 540))
painter := paint.New(paint.Dark())

func (g *Game) Update() error {
	keyboard.Update() // first, every tick

	if osk.IsKeyJustPressed(ebiten.KeyBackspace) { /* was inpututil.IsKeyJustPressed */ }
	text = osk.AppendInputChars(text)              // was ebiten.AppendInputChars

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	painter.Draw(screen, keyboard)
}
```

Try it:

```sh
go run github.com/bstkhq/go-ebiten-osk/examples/typing@latest
```

## How it is made

- **The keyboard draws nothing.** `osk.Keyboard` holds a layout and a rectangle, and turns the fingers and the mouse on its keys into key presses and typed text. Package `paint` draws it from a theme; a game with a look of its own draws it itself from `Keyboard.AppendKeys`.
- **A key reports its physical position**, as Ebitengine names positions, and types its character, so bindings by position and text fields both work. A key with no key of its own on the locale's physical board is text only.
- **Themes are values.** Start from `paint.Dark()` and change a colour, a size or the font.
- **A game reads the input copy.** `osk.IsKeyPressed`, `osk.AppendInputChars`, `osk.IsKeyJustPressed`, `osk.IsKeyJustReleased`, `osk.KeyPressDuration`, `osk.AppendPressedKeys`, `osk.AppendJustPressedKeys` and `osk.AppendJustReleasedKeys` are Ebitengine's and `inpututil`'s functions under another import. Why this, and not writing into the engine: [ADR 0001](docs/adr/0001-a-game-reads-the-input-copy.md).

The words used here and in the code are in the [glossary](GLOSSARY.md).

## State

Early. What works today:

- the compact en-US board (letters, space, erase), typed with fingers or the mouse;
- the input copy, alongside the physical keyboard;
- the stock painter with a dark theme.

Planned, in the [issues](https://github.com/bstkhq/go-ebiten-osk/issues): layers and the es-ES boards, the full form for names and emails, erase that repeats, a light theme and the full theme model, more locales, and an optional hook that makes Ebitengine's own functions report the on-screen keys, for code that cannot be changed.

Built and tested against Ebitengine v2.9.9. The API will change until the first tagged version.

## Tests

The suite needs a display, as anything that links Ebitengine does. With [go-ebiten-mcp](https://github.com/bstkhq/go-ebiten-mcp), on the GPU:

```sh
ebitenmcp run --gpu -- go test ./...
```

GitHub Actions uses a private display with software rendering and compares the golden pictures with the driver's tolerance. On Linux, install the [Ebitengine build dependencies](https://ebitengine.org/en/documents/install.html), Xvfb, and Mesa to run the same checks locally.

```sh
go build ./...
go vet ./...
go run github.com/bstkhq/go-ebiten-mcp/cmd/ebitenmcp run --x local --addr "" -- go test -count=1 ./...
```

With golangci-lint v2.15.0 installed and `origin/main` fetched, run the same advisory report and changed-line check as on a topic branch.

```sh
golangci-lint run --issues-exit-code=0
golangci-lint run --new-from-merge-base=origin/main
```

The changed-line check uses committed changes. On `main`, CI reports lint findings across the module.

The golden pictures under `testdata` were drawn by a hardware renderer, named beside each one: a software renderer may not match them to the pixel.

## Licence

MIT. See [LICENSE](LICENSE).
