package osk

import (
	"image"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
)

// A Keyboard is a board of keys on the screen. It holds a layout and the rectangle it
// occupies, and every [Keyboard.Update] turns the pointers on its keys into key presses and
// typed text for its input. It draws nothing.
type Keyboard struct {
	rect     image.Rectangle
	hidden   bool
	pointers Pointers
	input    *Input
	write    InputWriter

	// updated is the tick of the last update. A keyboard listens for landings from one
	// update to the next: a pointer it finds down after a tick without an update came down
	// while it was not listening, and presses nothing.
	updated int64

	// rows are the rows of the layer in view: the keyboard's own copy of them.
	rows [][]Key

	// keys are the keys of those rows, in their order, placed in the rectangle. positions is
	// the physical position each one presses, or noPosition, worked out once from the
	// layout; fingers counts the pointers each one holds.
	keys      []PlacedKey
	positions []ebiten.Key
	fingers   []int

	// landed is the key each pointer that is down landed on, or noKey.
	landed map[PointerID]int
	down   []Pointer

	// events is what the keyboard did in its last update, and changes what became of the
	// physical positions in it.
	events  []Event
	changes []change

	// held counts the keys pressed at each physical position: two keys of a layer may stand
	// for one position, which goes down with the first and up with the last.
	held [ebiten.KeyMax + 1]int
}

// A change is what became of a physical position in one update of a keyboard: whether the
// keyboard held it when the update began, and whether it went down in the update. With
// whether the keyboard holds it after, that tells a landing from a key let go and pressed
// again, and from one pressed and let go at once.
type change struct {
	key             ebiten.Key
	before, pressed bool
}

const (
	noKey      = -1
	noPosition = ebiten.Key(-1)
)

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
	// StateLatched is a key active until its effect is used once.
	StateLatched
	// StateLocked is a key active until it is toggled off.
	StateLocked
)

func (s State) String() string { return named("State", int(s), "idle", "pressed", "latched", "locked") }

// An Option changes how a keyboard is made.
type Option func(*options)

type options struct {
	pointers Pointers
	input    *Input
	write    InputWriter
}

// WithPointers makes the keyboard read its pointers from somewhere other than Ebitengine's
// touches and mouse. With nil it reads Ebitengine's.
func WithPointers(pointers Pointers) Option {
	return func(o *options) { o.pointers = pointers }
}

// WithInput makes the keyboard feed input instead of the default one, the one the package's
// own functions answer from. With nil it feeds the default one.
func WithInput(input *Input) Option {
	return func(o *options) { o.input = input }
}

// InputWriter receives a keyboard's held physical positions and events after
// each update. The events are valid only for the duration of the call.
type InputWriter func(held [ebiten.KeyMax + 1]bool, events []Event)

// WithInputWriter sends a keyboard's updates to writer in place of the input
// copy. A nil writer feeds the input copy as usual. The keyboard's events
// remain available through [Keyboard.AppendEvents].
func WithInputWriter(writer InputWriter) Option {
	return func(o *options) { o.write = writer }
}

// New is a keyboard of the layout, shown, on its first layer, with no rectangle yet. It
// refuses a layout that cannot work. The keyboard takes what it needs of the layout once,
// here: changing the one it was given afterwards changes nothing.
func New(layout Layout, opts ...Option) (*Keyboard, error) {
	if err := layout.check(); err != nil {
		return nil, err
	}

	var o options
	for _, opt := range opts {
		opt(&o)
	}

	if o.pointers == nil {
		o.pointers = &screen{}
	}

	if o.input == nil {
		o.input = defaultInput
	}

	k := &Keyboard{pointers: o.pointers, input: o.input, write: o.write, updated: never, landed: map[PointerID]int{}}

	for _, row := range layout.Layers[0].Rows {
		k.rows = append(k.rows, slices.Clone(row))

		for _, key := range row {
			k.positions = append(k.positions, key.Do.position(layout.Positions))
		}
	}

	k.keys = make([]PlacedKey, len(k.positions))
	k.fingers = make([]int, len(k.positions))
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
// types its text, until that pointer lifts; where it moves in between changes nothing.
//
// Call it once every tick, shown or hidden, before reading the input: an input answers for a
// keyboard only in the ticks it was updated in. A keyboard listens for landings from one
// update to the next. After a tick without an update, and on its first, it lets go of what it
// held, and a pointer it finds down presses nothing until it lifts and lands again, unless it
// says it landed in this very tick. A second update in one tick does nothing.
func (k *Keyboard) Update() {
	now := k.input.tick()
	if now == k.updated {
		return
	}

	listening := k.updated == now-1
	k.updated = now

	k.events = k.events[:0]
	k.changes = k.changes[:0]
	k.down = k.pointers.AppendPointers(k.down[:0])

	k.lift(listening)

	if !listening {
		// What it let go of was let go of while nobody was told: the game has seen
		// those keys go up already, and what comes now is no key pressed again.
		k.changes = k.changes[:0]
	}

	k.land(listening)
	k.deliver()
}

func (k *Keyboard) deliver() {
	if k.write == nil {
		k.input.feed(&k.held, k.changes, k.events)
		return
	}
	var held [ebiten.KeyMax + 1]bool
	for key, fingers := range k.held {
		held[key] = fingers > 0
	}
	k.write(held, k.events)
}

// lift releases the keys of the pointers that are no longer down, or that lifted and landed
// again since the last update; and of all of them while the keyboard is hidden or was not
// listening.
func (k *Keyboard) lift(listening bool) {
	for id, key := range k.landed {
		if !listening || k.hidden || !stayedDown(k.down, id) {
			k.release(key)
			delete(k.landed, id)
		}
	}
}

// stayedDown says whether the pointer is down and has been since before this tick.
func stayedDown(down []Pointer, id PointerID) bool {
	for _, pointer := range down {
		if pointer.ID == id {
			return !pointer.Landed
		}
	}

	return false
}

// land presses the keys under the pointers that came down in this tick: those that say so,
// and those the keyboard did not see in the tick before while it was listening. A pointer
// that came down while the keyboard was hidden or not listening, or outside every key, holds
// no key until it lifts. One that lifted again within the tick lets go at once.
func (k *Keyboard) land(listening bool) {
	for _, pointer := range k.down {
		if _, known := k.landed[pointer.ID]; known {
			continue
		}

		key := noKey
		if !k.hidden && (listening || pointer.Landed) {
			key = k.keyAt(pointer.X, pointer.Y)
		}

		k.press(key)

		if pointer.Lifted {
			k.release(key)
			key = noKey
		}

		k.landed[pointer.ID] = key
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

	if position := k.positions[key]; position != noPosition {
		if k.held[position] == 0 {
			k.changed(position).pressed = true
			k.events = append(k.events, Event{Kind: KeyDown, Key: position})
		}

		k.held[position]++
	}

	if text := k.keys[key].Do.text; text != "" {
		k.events = append(k.events, Event{Kind: Text, Text: text})
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

	if position := k.positions[key]; position != noPosition {
		if k.held[position] == 1 {
			k.changed(position)
			k.events = append(k.events, Event{Kind: KeyUp, Key: position})
		}

		k.held[position]--
	}
}

// changed is the change of the position in this update, begun now if it is its first: asked
// before the position goes down or up, so that it knows how the update found it.
func (k *Keyboard) changed(position ebiten.Key) *change {
	for i := range k.changes {
		if k.changes[i].key == position {
			return &k.changes[i]
		}
	}

	k.changes = append(k.changes, change{key: position, before: k.held[position] > 0})

	return &k.changes[len(k.changes)-1]
}

// place lays the rows out in the rectangle. What each key holds stays as it is.
func (k *Keyboard) place() {
	heaviest := 0.0
	for _, row := range k.rows {
		heaviest = math.Max(heaviest, rowWeight(row))
	}

	unit := float64(k.rect.Dx()) / heaviest
	placed := 0

	for i, row := range k.rows {
		top := k.rect.Min.Y + k.rect.Dy()*i/len(k.rows)
		bottom := k.rect.Min.Y + k.rect.Dy()*(i+1)/len(k.rows)
		left := float64(k.rect.Min.X) + (heaviest-rowWeight(row))*unit/2

		for _, key := range row {
			right := left + key.weight()*unit

			k.keys[placed].Key = key
			k.keys[placed].Rect = image.Rect(int(math.Round(left)), top, int(math.Round(right)), bottom)

			left = right
			placed++
		}
	}
}

func rowWeight(row []Key) float64 {
	total := 0.0
	for _, key := range row {
		total += key.weight()
	}

	return total
}
