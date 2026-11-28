// Package osk is an on-screen keyboard for Ebitengine that a game reads like a physical one.
//
// A [Keyboard] holds a [Layout] and the rectangle it occupies. Every tick it turns the fingers
// and the mouse on its keys into key presses and typed text. It draws nothing: package paint
// draws a keyboard from its state and a theme, and a game with a look of its own draws it
// itself from [Keyboard.AppendKeys] and [Keyboard.AppendVariants].
//
// A game reads the keyboard through the input copy: [IsKeyPressed], [AppendInputChars],
// [IsKeyJustPressed] and the others have the names and signatures of Ebitengine's functions
// and of inpututil's, and answer for the physical keyboard and the on-screen one together.
// Moving a game to the on-screen keyboard is a change of import:
//
//	keyboard.Update() // first in the game's Update
//
//	if osk.IsKeyJustPressed(ebiten.KeyBackspace) { ... }
//	text = osk.AppendInputChars(text)
//
// What the keyboard takes in during its Update is what the input copy answers until its next
// Update, so a game updates the keyboard every tick, shown or hidden, before it reads.
//
// A keyboard and its input belong to the game's Update, like the rest of a game's state: they
// are not safe for use from several goroutines.
package osk
