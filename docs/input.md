# Read keyboard input

Update each keyboard before reading its input. The input copy is the supported
interface for application code. Its functions have the same names and
signatures as the keyboard-reading functions in `ebiten` and `inpututil`.
They read both the physical keyboard and the on-screen keyboards together.

Use `osk.AppendInputChars` for text. Use `osk.IsKeyPressed`,
`osk.IsKeyJustPressed`, `osk.IsKeyJustReleased`, `osk.KeyPressDuration`,
`osk.AppendPressedKeys`, `osk.AppendJustPressedKeys`, and
`osk.AppendJustReleasedKeys` for keys. An on-screen key types its character
and reports its physical position as Ebitengine names it. Characters without
a physical position are text only.

By default, every keyboard feeds the package-level input copy. Use
`osk.NewInput()` and pass `osk.WithInput(input)` to `osk.New` when you need a
separate group of keyboards. Read that group's input through the methods on
`input`. A keyboard stops contributing held keys when you stop updating it.

See the [rationale for the input copy](adr/0001-a-game-reads-the-input-copy.md)
and the [glossary](../GLOSSARY.md).

## Touch and erase

Update the keyboard before handling the game's own pointers. Check
`keyboard.PointerTaken(osk.PointerID(id))` for an Ebitengine touch or
`keyboard.PointerTaken(osk.Mouse)` for the left mouse button. A pointer that
lands inside the shown keyboard belongs to it until it lifts, even if it lands
in blank space or slides outside. The first update after `Hide` releases its
keys. During either transition, a pointer that lands anywhere in the board's
rectangle is taken without pressing a key. Once fully hidden, the next update
clears ownership. A pointer that stays down needs to lift and land on the
fully shown keyboard to press a key.
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
and a rapid slide can look like a new landing. While a touch holds a letter
with variants, the stock source uses Ebitengine's touch press duration to
identify a new landing so the finger can slide to the picker. A custom
source that knows the edges can report them with `Pointer.Landed`.

The stock source matches a mouse landing to a touch within 12 logical pixels
in the current or three preceding ticks. The touch picks the key and the
matching mouse button owns no hold. The mouse counts as taken if either
landing belongs to the keyboard, including echoes just outside its rectangle.
When the mouse lands first, it selects the key immediately. A matching new touch one
tick later takes over the existing landing and controls its release. That
can keep the mouse's choice when the reports straddle two keys. An independent
mouse click in the same range and time window can look like an echo. Custom
pointer sources handle duplicate reports themselves.

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

Use the [demo's hook option](development.md#demo-controls) to try this mode.
