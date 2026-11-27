package osk

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// A Locale is the language and region a layout serves, named as in BCP 47.
type Locale string

// The locales that have stock layouts.
const (
	EnUS Locale = "en-US"
)

// A Form is how much keyboard is offered.
type Form int

const (
	// Compact is a small board: the letters and a few controls.
	Compact Form = iota
	// Full is enough to type names and emails. It is not a whole physical board.
	Full
)

func (f Form) String() string { return named("Form", int(f), "compact", "full") }

// named is the name of the value i of an enumeration, for a message a person reads: names[i],
// or what Go would print for a value that has none.
func named(enumeration string, i int, names ...string) string {
	if i < 0 || i >= len(names) || names[i] == "" {
		return fmt.Sprintf("%s(%d)", enumeration, i)
	}

	return names[i]
}

// A Layout is the keys of one locale in one form.
type Layout struct {
	// Layers are the faces of the layout, shown one at a time. The first is the one a
	// keyboard starts on.
	Layers []Layer

	// Positions says which characters have a key of their own on the locale's physical
	// board, and where. A key that types one of them also presses that position; every
	// other key is text only.
	Positions Positions
}

// A Layer is one face of a layout: its rows of keys, from the top.
type Layer struct {
	Rows [][]Key
}

// Positions maps a character to the physical position of the key that types it on a
// physical board, as Ebitengine names positions: by their place on a US board.
type Positions map[rune]ebiten.Key

// A Key is one touch target of a layer.
type Key struct {
	// Do is what the key does when it is pressed.
	Do Action

	// Label is what the key shows. A key with no label shows the text it types.
	Label string

	// Weight is the key's share of its row's width. Zero is one.
	Weight float64
}

// Face is what the key shows: its label, or the text it types when it has none.
func (k Key) Face() string {
	if k.Label != "" {
		return k.Label
	}

	return k.Do.text
}

// A Role is what a key is for, which decides how it is painted.
type Role int

const (
	// RoleCharacter is a key that types text.
	RoleCharacter Role = iota
	// RoleSpace is the key that types a space.
	RoleSpace
	// RoleControl is a key that acts on the text instead of adding to it.
	RoleControl
)

func (r Role) String() string { return named("Role", int(r), "character", "space", "control") }

// Role is what the key is for, told from what it does.
func (k Key) Role() Role {
	switch {
	case k.Do.kind == erases:
		return RoleControl
	case k.Do.text == " ":
		return RoleSpace
	default:
		return RoleCharacter
	}
}

func (k Key) weight() float64 {
	if k.Weight == 0 {
		return 1
	}

	return k.Weight
}

// An Action is what a key does. Its zero value does nothing, and a layout with such a key is
// refused: make one with [Type] or [Erase].
type Action struct {
	kind actionKind
	text string
}

type actionKind int

const (
	types actionKind = iota + 1
	erases
)

// Type is the action of a key that types text: a character, usually, or a space.
func Type(text string) Action {
	return Action{kind: types, text: text}
}

// Erase is the action of the key that erases what was typed. It presses backspace and types
// nothing.
func Erase() Action {
	return Action{kind: erases}
}

// position is the physical position the action presses, or noPosition for one that is text
// only: its text is not one character, or the layout gives that character no position that
// Ebitengine has.
func (a Action) position(positions Positions) ebiten.Key {
	if a.kind == erases {
		return ebiten.KeyBackspace
	}

	runes := []rune(a.text)
	if len(runes) != 1 {
		return noPosition
	}

	key, ok := positions[runes[0]]
	if !ok || key < 0 || key > ebiten.KeyMax {
		return noPosition
	}

	// A modifier named for either side has no key of its own: it stands for the left one.
	if both, named := sides[key]; named {
		key = both[0]
	}

	return key
}

// check says why a keyboard cannot work with the layout, or nil.
func (l Layout) check() error {
	if len(l.Layers) == 0 {
		return fmt.Errorf("osk: the layout has no layer")
	}

	for i, layer := range l.Layers {
		if len(layer.Rows) == 0 {
			return fmt.Errorf("osk: layer %d has no row", i+1)
		}

		for j, row := range layer.Rows {
			if err := checkRow(row); err != nil {
				return fmt.Errorf("osk: layer %d, row %d: %w", i+1, j+1, err)
			}
		}
	}

	return nil
}

func checkRow(row []Key) error {
	if len(row) == 0 {
		return fmt.Errorf("it has no key")
	}

	for i, key := range row {
		switch {
		case key.Do.kind == 0:
			return fmt.Errorf("key %d does nothing", i+1)
		case key.Do.kind == types && key.Do.text == "":
			return fmt.Errorf("key %d types nothing", i+1)
		case key.Weight < 0 || math.IsNaN(key.Weight) || math.IsInf(key.Weight, 0):
			return fmt.Errorf("key %d has the weight %v", i+1, key.Weight)
		}
	}

	return nil
}
