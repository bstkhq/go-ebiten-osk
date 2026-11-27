package osk

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// A Keyboard is a board of keys on the screen. It holds a layout and the rectangle it
// occupies, and every [Keyboard.Update] turns the pointers on its keys into key presses and
// typed text for its input. It draws nothing.
type Keyboard struct {
	layout   Layout
	rect     image.Rectangle
	hidden   bool
	pointers Pointers
	input    *Input

	// keys are the keys of the layer in view, placed in the rectangle. fingers counts the
	// pointers each one holds.
	keys    []PlacedKey
	fingers []int

	// landed is the key each pointer that is down landed on, or noKey.
	landed map[PointerID]int
	down   []Pointer

	events  []Event
	typed   []rune
	presses presses

	// updated is the tick of the last update: an input stops hearing a keyboard that is no
	// longer updated.
	updated int64
}

const noKey = -1

// A PlacedKey is a key as its keyboard holds it now: where it is and how it looks.
type PlacedKey struct {
	Key

	// Rect is the key's place on the screen. The keys of a row tile it with no gap between
	// them: a painter insets them as its theme says.
	Rect image.Rectangle

	State State
}

// A State is how a key looks right now.
type State int

const (
	// StateIdle is a key nothing is holding.
	StateIdle State = iota
	// StatePressed is a key under a finger or the mouse.
	StatePressed
)

func (s State) String() string { return named("State", int(s), "idle", "pressed") }

// An Option changes how a keyboard is made.
type Option func(*options)

type options struct {
	pointers Pointers
	input    *Input
}

// WithPointers makes the keyboard read its pointers from somewhere other than Ebitengine's
// touches and mouse.
func WithPointers(pointers Pointers) Option {
	return func(o *options) { o.pointers = pointers }
}

// WithInput makes the keyboard feed input instead of the default one, the one the package's
// own functions answer from.
func WithInput(input *Input) Option {
	return func(o *options) { o.input = input }
}

// New is a keyboard of the layout, shown, on its first layer, with no rectangle yet. It
// refuses a layout that cannot work.
func New(layout Layout, opts ...Option) (*Keyboard, error) {
	if err := layout.check(); err != nil {
		return nil, err
	}

	o := options{pointers: &screen{}, input: defaultInput}
	for _, opt := range opts {
		opt(&o)
	}

	k := &Keyboard{layout: layout, pointers: o.pointers, input: o.input, landed: map[PointerID]int{}}
	k.place()

	return k, nil
}

// SetRect gives the keyboard the rectangle it occupies. Its keys fill it: the rows share its
// height, and the keys of a row share its width by weight. A row lighter than the heaviest is
// centred.
func (k *Keyboard) SetRect(rect image.Rectangle) {
	k.rect = rect
	k.place()
}

// Rect is the rectangle the keyboard occupies.
func (k *Keyboard) Rect() image.Rectangle {
	return k.rect
}

// Show brings the keyboard back after [Keyboard.Hide].
func (k *Keyboard) Show() {
	k.hidden = false
}

// Hide takes the keyboard away. From its next update it takes no pointer, and the keys it
// held are released.
func (k *Keyboard) Hide() {
	k.hidden = true
}

// Shown says whether the keyboard is on the screen.
func (k *Keyboard) Shown() bool {
	return !k.hidden
}

// AppendKeys appends the keys of the layer in view, as the keyboard holds them now, and
// returns the extended slice. It is what a painter draws.
func (k *Keyboard) AppendKeys(keys []PlacedKey) []PlacedKey {
	return append(keys, k.keys...)
}

// AppendEvents appends what the keyboard did in its last update, in the order it happened,
// and returns the extended slice.
func (k *Keyboard) AppendEvents(events []Event) []Event {
	return append(events, k.events...)
}

// Update takes in the pointers of this tick. A pointer that lands on a key presses it, and
// types its text, until that pointer lifts; where it moves in between changes nothing. Call
// it once every tick, shown or hidden, before reading the input: an input stops hearing a
// keyboard that is no longer updated, as if every key it held had vanished.
func (k *Keyboard) Update() {
	k.events = k.events[:0]
	k.typed = k.typed[:0]
	k.down = k.pointers.AppendPointers(k.down[:0])

	k.lift()
	k.land()
	k.presses.tick()
	k.input.hear(k)
}

// lift releases the keys of the pointers that are no longer down, and of all of them while
// the keyboard is hidden.
func (k *Keyboard) lift() {
	for id, key := range k.landed {
		if k.hidden || !isDown(k.down, id) {
			k.release(key)
			delete(k.landed, id)
		}
	}
}

func isDown(down []Pointer, id PointerID) bool {
	for _, pointer := range down {
		if pointer.ID == id {
			return true
		}
	}

	return false
}

// land presses the keys under the pointers that came down in this tick. A pointer that came
// down while the keyboard was hidden, or outside every key, holds no key until it lifts.
func (k *Keyboard) land() {
	for _, pointer := range k.down {
		if _, known := k.landed[pointer.ID]; known {
			continue
		}

		key := noKey
		if !k.hidden {
			key = k.keyAt(pointer.X, pointer.Y)
		}

		k.landed[pointer.ID] = key
		k.press(key)
	}
}

func (k *Keyboard) keyAt(x, y int) int {
	at := image.Pt(x, y)

	for i, key := range k.keys {
		if at.In(key.Rect) {
			return i
		}
	}

	return noKey
}

// press puts one more pointer on a key. The first one presses it.
func (k *Keyboard) press(key int) {
	if key == noKey {
		return
	}

	k.fingers[key]++
	if k.fingers[key] > 1 {
		return
	}

	k.keys[key].State = StatePressed
	action := k.keys[key].Do

	if position, ok := action.position(k.layout.Positions); ok {
		k.presses.held[position]++
		k.events = append(k.events, Event{Kind: KeyDown, Key: position})
	}

	if action.text != "" {
		k.typed = append(k.typed, []rune(action.text)...)
		k.events = append(k.events, Event{Kind: Text, Text: action.text})
	}
}

// release takes one pointer off a key. The last one releases it.
func (k *Keyboard) release(key int) {
	if key == noKey {
		return
	}

	k.fingers[key]--
	if k.fingers[key] > 0 {
		return
	}

	k.keys[key].State = StateIdle

	if position, ok := k.keys[key].Do.position(k.layout.Positions); ok {
		k.presses.held[position]--
		k.events = append(k.events, Event{Kind: KeyUp, Key: position})
	}
}

// place lays the first layer out in the rectangle, keeping what each key holds.
func (k *Keyboard) place() {
	rows := k.layout.Layers[0].Rows

	states := k.keys
	k.keys = k.keys[len(k.keys):]

	heaviest := 0.0
	for _, row := range rows {
		heaviest = math.Max(heaviest, rowWeight(row))
	}

	for i, row := range rows {
		top := k.rect.Min.Y + k.rect.Dy()*i/len(rows)
		bottom := k.rect.Min.Y + k.rect.Dy()*(i+1)/len(rows)
		unit := float64(k.rect.Dx()) / heaviest
		left := float64(k.rect.Min.X) + (heaviest-rowWeight(row))*unit/2

		for _, key := range row {
			right := left + key.weight()*unit
			k.keys = append(k.keys, PlacedKey{
				Key:  key,
				Rect: image.Rect(int(math.Round(left)), top, int(math.Round(right)), bottom),
			})
			left = right
		}
	}

	for i := range states {
		k.keys[i].State = states[i].State
	}

	if k.fingers == nil {
		k.fingers = make([]int, len(k.keys))
	}
}

func rowWeight(row []Key) float64 {
	total := 0.0
	for _, key := range row {
		total += key.weight()
	}

	return total
}

// presses is what a keyboard holds of each physical position, as its input reads it.
type presses struct {
	// held counts the keys pressed at each position.
	held [ebiten.KeyMax + 1]int
	// ticks counts the updates each position has been pressed for; zero is up.
	ticks [ebiten.KeyMax + 1]int
	// was says whether each position was pressed after the update before the last.
	was [ebiten.KeyMax + 1]bool
}

func (p *presses) tick() {
	for position := range p.held {
		p.was[position] = p.ticks[position] > 0

		if p.held[position] > 0 {
			p.ticks[position]++
		} else {
			p.ticks[position] = 0
		}
	}
}
