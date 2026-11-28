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

Press F2 to cycle between the dark theme, the light theme, and the example's own painter.

## Layouts

In the typing example, press F1 for compact en-US, F3 for compact es-ES, F5 for
full en-US, or F6 for full es-ES. F4 shows a PIN pad built from the exported
layout blocks.

`osk.Stock` looks up a layout by locale and form. Compact `osk.EnUS` and `osk.EsES`
both have uppercase letters, a `123` layer with digits and punctuation, and an
`ABC` switch back to letters. Spanish also has an `ÁÉÍ` layer.

Call `Keyboard.SetLayout` to change form or locale or use a custom layout. Its first
layer takes effect immediately, and held keys are released in the next
`Keyboard.Update`. Call it before `Update` to read those releases in the same
game tick. Custom layouts use a `Layer.Name` and a key with
`osk.SwitchLayer(name)` to link layers.

### Full form

Ask `osk.Stock` for `osk.Full` to start with lowercase letters and a localized
Enter key. Tap Shift once for one capital, twice for caps lock, and a third time
to unlock. Spanish supports both cases on the accents layer. The `123` layer
has digits and symbols.

Use `osk.StockWithAction(osk.EsES, osk.Full, osk.ActionNext)` to label the action
key «Siguiente». Enter, Next, Done, Go, Search, and Send all press `ebiten.KeyEnter`
and emit `osk.ActionPressed`. Read `event.FieldAction` from `Keyboard.AppendEvents`
to handle the selected action. Custom layouts can use `osk.Shift()` and
`osk.Submit(...)` for these keys.

## Customize the keyboard

Copy the default theme and change the fields you need. Sizes are fractions of a row's height.

```go
theme := paint.Dark()
theme.Font = myFont // *sfnt.Font, for example from sfnt.Parse
theme.CharacterSize = 0.38
theme.Palette.Key = color.RGBA{R: 0x20, G: 0x40, B: 0x60, A: 0xff}
painter.SetTheme(theme) // takes effect on the next Draw
```

The palette sets the panel, character keys, control keys, action key, and labels. A palette edit
also updates the derived colours, including the pressed face and the lip. Use
`theme.Roles[osk.RoleControl].All` for every state of a role, `theme.States[osk.StatePressed]`
for every role in a state, or `theme.Roles[osk.RoleControl].States[osk.StatePressed]` for a
specific combination. Each override can set any of `Face`, `Lip`, `Border`, and `Label`.
Leave a colour nil to inherit it, or use `color.Transparent` to make it transparent.

Set a layout key's `Icon` to `osk.IconShift`, `osk.IconErase`, or `osk.IconAction` to
draw a shape with the stock painter. Keep a label for painters that use words.

To replace the painter, call your own drawing code in the game's `Draw`. Use
`keyboard.Shown()` and `keyboard.Rect()` for the panel, and `keyboard.AppendKeys` for each
key's rectangle, role, state, and label or icon. The example's
[custom painter](examples/typing/painter.go) uses this same public state.

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
in blank space or slides outside. After hiding the keyboard, its next update
releases its keys and clears ownership. A pointer that stays down while the
keyboard is shown again needs to lift and land to press a key.
Changing the layer or layout releases the old keys and keeps their pointers
taken until lift.

An erase press sends Backspace and repeats after 30 game ticks, then every
6 ticks. At 60 ticks per second these are 500 ms and 100 ms. Set the two counts
with `osk.WithEraseRepeat(delay, interval)`, or pass zero for either to disable
repeat. Counting ticks lets a game that handles `IsKeyJustPressed` erase once
per repeat with deterministic timing at its chosen tick rate.

The stock pointer source sees one snapshot per tick. When a touch ID moves
more than 48 logical pixels between consecutive samples, the keyboard treats
it as a lift and a new landing. That catches reuse between adjacent letter
centres on a 640-pixel compact board. A closer reuse can look like a slide,
and a rapid slide can look like a new landing. A custom source that knows the
edges can report them with `Pointer.Landed`.

The stock source matches a mouse landing to a touch within 12 logical pixels
in the current or three preceding ticks. The touch picks the key and the
matching mouse button owns no hold. The mouse counts as taken if either
landing belongs to the keyboard, including echoes just outside its rectangle.
When the mouse lands first, it presses immediately. A matching new touch one
tick later takes over the existing landing and controls its release. That
can keep the mouse's choice when the reports straddle two keys. An independent
mouse click in the same range and time window can look like an echo. Custom
pointer sources handle duplicate reports themselves.

## State

Early. What works today:

- compact and full en-US and es-ES boards, typed with fingers or the mouse;
- the input copy, alongside the physical keyboard;
- the stock painter with dark and light themes.

Planned, in the [issues](https://github.com/bstkhq/go-ebiten-osk/issues): email and number kinds, more locales, and an optional hook that makes Ebitengine's own functions report the on-screen keys, for code that cannot be changed.

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

When the checkout contains tests in `hook/internal/upstream`, run the declaration check separately with no display.

```sh
env -u DISPLAY -u WAYLAND_DISPLAY go test -count=1 ./hook/internal/upstream
```

When `hook/nohook.go` is present, also run the suite with the hook compiled out.

```sh
go run github.com/bstkhq/go-ebiten-mcp/cmd/ebitenmcp run --x local --addr "" -- go test -count=1 -tags ebitenosk_nohook ./...
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
