package osk

import (
	"fmt"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// A Locale is the language and region a layout serves, named as in BCP 47.
type Locale string

// The locales that have stock layouts.
const (
	EnUS Locale = "en-US"
	EsES Locale = "es-ES"
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

// A Layout defines the layers and physical positions of a keyboard. A stock layout
// serves a locale, form and kind.
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
	// Name is how a layer switch refers to this layer. A layer can be unnamed if
	// nothing switches to it. Nonempty names must be unique within the layout.
	Name string
	Rows [][]Key

	// RowGaps reserves blank space in units of [Key.Weight]. When supplied,
	// it has one entry per row. A nil entry leaves that row contiguous.
	// Otherwise the entry has len(row)+1 finite, nonnegative weights, before
	// the first key, between keys, and after the last key.
	RowGaps [][]float64
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

	// Icon selects a shape to show in place of the label. The label remains
	// available to painters that do not draw the icon.
	Icon Icon

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
	// RoleLayerSwitch is a key that takes the keyboard to a layer.
	RoleLayerSwitch
	// RoleAction is a key that completes the current field.
	RoleAction
)

func (r Role) String() string {
	return named("Role", int(r), "character", "space", "control", "layer switch", "action")
}

// Role is what the key is for, told from what it does.
func (k Key) Role() Role {
	switch {
	case k.Do.kind == switchesLayer:
		return RoleLayerSwitch
	case k.Do.kind == erases, k.Do.kind == shifts:
		return RoleControl
	case k.Do.kind == submits:
		return RoleAction
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
// refused: make one with [Type], [Erase], [SwitchLayer], [Shift] or [Submit].
type Action struct {
	kind   actionKind
	text   string
	target string
	field  FieldAction
}

type actionKind int

const (
	types actionKind = iota + 1
	erases
	switchesLayer
	shifts
	submits
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

// SwitchLayer is the action of a key that shows the layer with this name. It types nothing
// and reports no press. Give the key a Label for its face.
func SwitchLayer(name string) Action {
	return Action{kind: switchesLayer, target: name}
}

// position is the physical position the action presses, or noPosition for one that is text
// only: its text is not one character, or the layout gives that character no position that
// Ebitengine has.
func (a Action) position(positions Positions) ebiten.Key {
	switch a.kind {
	case erases:
		return ebiten.KeyBackspace
	case submits:
		return ebiten.KeyEnter
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

const noLayer = -1

// resolve takes a keyboard's own copy of the rows and looks up what each key presses
// or which layer it shows. An invalid layout leaves the keyboard untouched.
func (l Layout) resolve() ([]resolvedLayer, error) {
	if len(l.Layers) == 0 {
		return nil, fmt.Errorf("osk: the layout has no layer")
	}

	names, err := layerNames(l.Layers)
	if err != nil {
		return nil, err
	}

	layers := make([]resolvedLayer, len(l.Layers))
	for i, layer := range l.Layers {
		layers[i], err = resolveLayer(layer, l.Positions, names)
		if err != nil {
			return nil, fmt.Errorf("osk: layer %d: %w", i+1, err)
		}
	}

	return layers, nil
}

func layerNames(layers []Layer) (map[string]int, error) {
	names := make(map[string]int, len(layers))
	for i, layer := range layers {
		if layer.Name == "" {
			continue
		}
		if _, exists := names[layer.Name]; exists {
			return nil, fmt.Errorf("osk: layer %d repeats the name %q", i+1, layer.Name)
		}
		names[layer.Name] = i
	}

	return names, nil
}

// A resolved layer holds its own keys so a pointer can refer to the key it landed on
// even after SetLayout has made another layer visible.
type resolvedLayer struct {
	rows []resolvedRow
}

type resolvedRow struct {
	keys   []keyState
	weight float64
}

type keyState struct {
	PlacedKey
	position   ebiten.Key
	target     int
	fingers    int
	capital    string
	leadingGap float64
}

func resolveLayer(layer Layer, positions Positions, names map[string]int) (resolvedLayer, error) {
	if len(layer.Rows) == 0 {
		return resolvedLayer{}, fmt.Errorf("it has no row")
	}
	if len(layer.RowGaps) != 0 && len(layer.RowGaps) != len(layer.Rows) {
		return resolvedLayer{}, fmt.Errorf("row gaps must describe each row")
	}

	rows := make([]resolvedRow, len(layer.Rows))
	for i, row := range layer.Rows {
		var gaps []float64
		if len(layer.RowGaps) != 0 {
			gaps = layer.RowGaps[i]
		}
		resolved, err := resolveRow(row, gaps, positions, names)
		if err != nil {
			return resolvedLayer{}, fmt.Errorf("row %d: %w", i+1, err)
		}
		rows[i] = resolved
	}

	return resolvedLayer{rows: rows}, nil
}

func resolveRow(row []Key, gaps []float64, positions Positions, names map[string]int) (resolvedRow, error) {
	if len(row) == 0 {
		return resolvedRow{}, fmt.Errorf("it has no key")
	}
	weight, err := gapWeight(gaps, len(row))
	if err != nil {
		return resolvedRow{}, err
	}

	resolved := resolvedRow{keys: make([]keyState, len(row)), weight: weight}
	for i, key := range row {
		state, err := resolveKey(key, positions, names)
		if err != nil {
			return resolvedRow{}, fmt.Errorf("key %d %w", i+1, err)
		}
		if gaps != nil {
			state.leadingGap = gaps[i]
		}
		resolved.keys[i] = state
		resolved.weight += key.weight()
	}

	return resolved, nil
}

func gapWeight(gaps []float64, keyCount int) (float64, error) {
	if gaps == nil {
		return 0, nil
	}
	if len(gaps) != keyCount+1 {
		return 0, fmt.Errorf("it needs one more gap than keys")
	}
	var total float64
	for _, gap := range gaps {
		if !validWeight(gap) {
			return 0, fmt.Errorf("has invalid gap weight %v", gap)
		}
		total += gap
	}
	return total, nil
}

func validWeight(weight float64) bool {
	return weight >= 0 && !math.IsInf(weight, 0)
}

func resolveKey(key Key, positions Positions, names map[string]int) (keyState, error) {
	if err := key.Do.validate(); err != nil {
		return keyState{}, err
	}
	if !validWeight(key.Weight) {
		return keyState{}, fmt.Errorf("has the weight %v", key.Weight)
	}

	state := keyState{
		PlacedKey: PlacedKey{Key: key}, position: key.Do.position(positions), target: noLayer,
		capital: strings.ToUpper(key.Do.text),
	}
	if key.Do.kind == switchesLayer {
		target, ok := names[key.Do.target]
		if !ok {
			return keyState{}, fmt.Errorf("switches to an unknown layer %q", key.Do.target)
		}
		state.target = target
	}

	return state, nil
}

func (a Action) validate() error {
	switch {
	case a.kind == 0:
		return fmt.Errorf("does nothing")
	case a.kind == types && a.text == "":
		return fmt.Errorf("types nothing")
	case a.kind == submits && !a.field.valid():
		return fmt.Errorf("has unknown field action %v", a.field)
	default:
		return nil
	}
}
