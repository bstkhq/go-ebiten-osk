# go-ebiten-osk

An on-screen keyboard for Ebitengine that a game reads like a physical one. These are the words its code, issues and documents use.

## Language

**Keyboard**:
The board of keys drawn on the screen that a person types on by touch or with the mouse.
_Avoid_: OSK (the package's name only), soft keyboard, virtual keyboard

**Locale**:
The language and region a layout serves, named as in BCP 47 (`es-ES`).
_Avoid_: language, country

**Form**:
How much keyboard is offered: compact, a small board of letters and a few controls, or full, enough for names and emails.
_Avoid_: size, mode, variant

**Layout**:
The layers and physical positions of a keyboard. A stock layout serves one locale in one form.
_Avoid_: keymap, board definition

**Layer**:
One face of a layout, shown at a time: the letters, the digits and symbols, the accents.
_Avoid_: page, view, level

**Layer switch**:
A key that shows a named layer of the layout.

**Key**:
One touch target of a layer. It types text, erases, or changes what the keyboard shows.
_Avoid_: button, cap

**Role**:
What a key is for. The roles are character, space, control, layer switch and action.
_Avoid_: type, class

**Kind**:
What the text being typed is: text, an email, a number.
_Avoid_: input type, mode

**Physical position**:
The key of a physical keyboard that an on-screen key stands for, named as Ebitengine names it, by its place on a US board. Ñ on a Spanish board is the position called semicolon.
_Avoid_: key code, scancode

**Pointer**:
One finger on the screen, or the mouse with its button down.
_Avoid_: touch, cursor, when either could be meant

**Input copy**:
The library's functions with the names and signatures of Ebitengine's keyboard-reading functions, which answer for the physical keyboard and the on-screen one together.
_Avoid_: wrapper, shim

**Hook**:
The optional writer of on-screen keys into Ebitengine's own input state, for code that cannot be changed.
_Avoid_: injector, patch

**Theme**:
The values that give a keyboard its look: colours, sizes, fonts.
_Avoid_: skin, style

**Palette**

The base colours of a theme, used to derive the colours of keys in each state.

**Lip**

The lower edge visible beneath a raised key.

**Painter**:
What draws a keyboard from its state and a theme.
_Avoid_: renderer, view
