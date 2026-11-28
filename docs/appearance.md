# Draw and animate the keyboard

The keyboard tracks input and layout. The `paint` package draws its current
state with a theme. Start with `paint.Dark()` or `paint.Light()`.

## Themes

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

## Custom painters

To replace the painter, call your own drawing code in the game's `Draw`. Use
`keyboard.Visibility()` and `keyboard.Rect()` for the panel and
`keyboard.AppendKeys` for each key's rectangle, role, state, and label or icon.
Draw the accent picker last using `keyboard.AppendVariants`. Its key rectangles
can extend above the keyboard rectangle. The demo's
[custom painter](../examples/typing/painter.go) uses this same public state.

## Show and hide

A new keyboard starts fully shown. `Show` and `Hide` animate by default over
200 ms (12 updates at 60 ticks per second). The stock painter slides and fades
the whole board within its rectangle. Change the duration with
`osk.WithAnimationDuration(duration)` at construction or
`keyboard.SetAnimationDuration(duration)` at runtime. A positive duration is
rounded up to whole updates at the game's configured tick rate when a
transition starts. Without a positive tick rate, the calculation uses 60.
A duration of zero or less disables animation and finishes any transition
immediately.

Read `keyboard.Visibility()` for the eased fraction from 0 to 1. Use it to
move the game's layout or to animate a custom painter. `keyboard.Shown()`
stays true until a hide finishes. Calling `Show` during a hide or `Hide` during
a show reverses the current transition from its present position.

See [pointer ownership during transitions](input.md#touch-and-erase).
