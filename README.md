# go-ebiten-osk

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

## Touch and erase

Update the keyboard before handling the game's own pointers. Check
`keyboard.PointerTaken(osk.PointerID(id))` for an Ebitengine touch or
`keyboard.PointerTaken(osk.Mouse)` for the left mouse button. A pointer that
lands inside the shown keyboard belongs to it until it lifts, even if it lands
in blank space or slides outside. Hiding the keyboard releases its keys and
clears ownership. A pointer that stays down while the keyboard is shown again
needs to lift and land to press a key.

An erase press sends Backspace and repeats after 30 game ticks, then every
6 ticks. At 60 ticks per second these are 500 ms and 100 ms. Set the two counts
with `osk.WithEraseRepeat(delay, interval)`, or pass zero for either to disable
repeat. Counting ticks lets a game that handles `IsKeyJustPressed` erase once
per repeat with deterministic timing at its chosen tick rate.

The stock pointer source sees one snapshot per tick. When a touch ID moves
more than 64 logical pixels between consecutive samples, the keyboard treats
it as a lift and a new landing. A close reuse of the same ID can look like a
slide, and a fast enough slide can look like a new landing. A custom source
that knows the edges can report them with `Pointer.Landed`.

The stock source also handles touchscreens that echo a touch as a mouse press.
A mouse landing on a key waits one tick for a touch within 12 logical pixels.
Touches in the preceding, same or following tick can match. The touch chooses
the key, and a matching mouse press stays suppressed until the button lifts.
A genuine mouse press in that range and time window can look like an echo.
Stock mouse landings are taken immediately for `PointerTaken`. Custom pointer
sources supply their own edges and handle any duplicate reports themselves.

## State

Early. What works today:

- the compact en-US board (letters, space, erase), typed with fingers or the mouse;
- the input copy, alongside the physical keyboard;
- the stock painter with a dark theme.

Planned, in the [issues](https://github.com/bstkhq/go-ebiten-osk/issues): layers and the es-ES boards, the full form for names and emails, a light theme and the full theme model, more locales, and an optional hook that makes Ebitengine's own functions report the on-screen keys, for code that cannot be changed.

Built and tested against Ebitengine v2.9.9. The API will change until the first tagged version.

## Tests

The suite needs a display, as anything that links Ebitengine does. With [go-ebiten-mcp](https://github.com/bstkhq/go-ebiten-mcp), on the GPU:

```sh
ebitenmcp run --gpu -- go test ./...
```

The golden pictures under `testdata` were drawn by a hardware renderer, named beside each one: a software renderer may not match them to the pixel.

## Licence

MIT. See [LICENSE](LICENSE).
