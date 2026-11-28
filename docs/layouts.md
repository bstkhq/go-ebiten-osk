# Choose and customize layouts

## Stock locales

`osk.StockLocales` lists the available locales. `osk.Stock` looks up a layout by
locale and form. Compact US English (`osk.EnUS`) and Spanish (`osk.EsES`)
both have uppercase letters, a `123` layer with digits and punctuation, and
an `ABC` switch back to letters.

British English (`osk.EnGB`) includes £ on its symbols layer. German
(`osk.DeDE`) puts Ä, Ö, Ü and ß on its QWERTZ letters layer. Italian
(`osk.ItIT`) has QWERTY letters. Compact Italian shows capitals, and full
Italian switches case with Shift.

French (`osk.FrFR`, traditional AZERTY), Portuguese (`osk.PtPT`), and Catalan
(`osk.CaES`) support both forms and all field kinds, with localized controls
and field actions.

The locale tables are written by hand. Physical positions were checked against
the default boards in [xkeyboard-config](https://xkeyboard-config.freedesktop.org/doc/symbols/).

## Accent picker

Spanish, French, Portuguese, Catalan, and Italian offer accented letters in a
picker in both forms. Tap for the base letter. To choose a variant, hold the
base key for about half a second, slide to the variant, and release. Release
outside the picker to cancel. The picker follows Shift and caps lock in the
full form. Catalan's middle dot (·) has its own key on the letters layer.

## Custom layouts

Call `Keyboard.SetLayout` to change form or locale or use a custom layout. Its first
layer takes effect immediately, and held keys are released in the next
`Keyboard.Update`. Call it before `Update` to read those releases in the same
game tick. Custom layouts use a `Layer.Name` and a key with
`osk.SwitchLayer(name)` to link layers. Use `Layer.RowGaps` to reserve blank
space before, between, or after a row's keys in the same units as their weights.

## Full form and field actions

Ask `osk.Stock` for `osk.Full` to start with lowercase letters and a return (↵)
key. Tap Shift once for one capital, twice for caps lock, and a third time
to unlock. The `123` layer
has digits and symbols.

Use `osk.StockWithAction(osk.EsES, osk.Full, osk.ActionNext)` to label the action
key «Siguiente». Enter, Next, Done, Go, Search, and Send all press `ebiten.KeyEnter`
and emit `osk.ActionPressed`. Read `event.FieldAction` from `Keyboard.AppendEvents`
to handle the selected action. Custom layouts can use `osk.Shift()` and
`osk.Submit(...)` for these keys. Set a custom action key's `Label` to give it
your own text.

Pass `osk.WithActionHandler(func(action osk.FieldAction) { ... })` to `osk.New`
to receive each on-screen action directly. The handler runs at the end of
`Keyboard.Update` and can change the layout or hide the keyboard. The same
activation still reports an event and presses Enter. If your form also reads
Enter through the input copy or hook, skip the Enter handler when the callback
already supplied an action in that update. Physical Enter continues to come
through the input API.

## Field kinds

Use `osk.StockWithKind` to select the keys for a field. For example, set the
layout for an email field with a Next action.

```go
layout, err := osk.StockWithKind(osk.EnUS, osk.Full, osk.KindEmail, osk.ActionNext)
if err != nil {
	return err
}
return keyboard.SetLayout(layout)
```

`osk.KindText` gives the same layout as `osk.StockWithAction`. On the full
letters layer, Text has a comma immediately to the left of Space and Email
has **@** in the same slot. The period stays on the right. Space spans four
letter slots, and `123`, Shift, erase and the field action span one and a half
slots each. Extra symbols sit beside Space on the digits and symbols layer
to keep erase at the same width. Short rows leave space around keys.
German uses eleven slots across the letters layer, while Catalan and
British English center ten-slot rows on an eleven-slot grid to fit longer
rows elsewhere.

Compact email uses the compact text layout.
`osk.KindNumber` gives both forms a localized pad with ten digits, erase, and
the selected field action. Digits report the physical digit positions.

See [how to read input](input.md) and [how to draw a custom layout](appearance.md#custom-painters).
