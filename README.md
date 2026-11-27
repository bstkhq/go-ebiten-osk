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

## Optional Ebitengine hook

Use `github.com/bstkhq/go-ebiten-osk/hook` for code that already reads
`ebiten` and `inpututil` directly, such as a toolkit you cannot change.
It writes keys and characters into Ebitengine's internal input snapshot.
The hook targets Ebitengine 2.9.x releases and is tested against v2.9.9.
Each engine minor version needs its own verification.

During setup, call `hook.New()` and pass the resulting hook's `Option()` to
`osk.New`. If `hook.New()` returns an error, omit that option to use the input
copy. In the game's `Update`, update the hooked keyboards first, then call
`h.Update()` before any keyboard reads or updates of keyboards that feed the
input copy. Keep calling `h.Update()` each tick so that a keyboard you stop
updating releases its keys. The input copy sees hooked keys and characters
through Ebitengine, and `Keyboard.AppendEvents` is still available.

A physically held key keeps its own Ebitengine state. A physical press that
takes over an on-screen hold can report a new press and restart the duration.
Use the input copy without the hook for a continuous hold across both sources.

Importing the hook opts into linking its engine integration. With
`-tags ebitenosk_nohook`, the package compiles without the internal symbols
and `hook.New()` reports that the hook is disabled. Use this tag when building
against an engine version that has removed one of those symbols.

```sh
go run github.com/bstkhq/go-ebiten-osk/examples/typing@latest -hook
```

## State

Early. What works today:

- the compact en-US board (letters, space, erase), typed with fingers or the mouse;
- the input copy, alongside the physical keyboard;
- the stock painter with a dark theme;
- the optional hook for Ebitengine 2.9.x.

Planned, in the [issues](https://github.com/bstkhq/go-ebiten-osk/issues): layers and the es-ES boards, the full form for names and emails, erase that repeats, a light theme and the full theme model, and more locales.

Built and tested against Ebitengine v2.9.9. The API will change until the first tagged version.

## Tests

The game tests need a display. With [go-ebiten-mcp](https://github.com/bstkhq/go-ebiten-mcp), on the GPU:

```sh
ebitenmcp run --gpu -- go test ./...
```

The golden pictures under `testdata` were drawn by a hardware renderer, named beside each one: a software renderer may not match them to the pixel.

Run `go test ./hook/internal/upstream` without a display to check the engine declarations the hook relies on.

## Licence

MIT. See [LICENSE](LICENSE).
