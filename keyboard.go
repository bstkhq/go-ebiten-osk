package osk

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

// A Keyboard is a board of keys on the screen. It holds a layout and the rectangle it
// occupies, and every [Keyboard.Update] turns the pointers on its keys into key presses and
// typed text for its input. It draws nothing. Make one with [New].
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

	layers   []resolvedLayer
	layer    int
	shift    State
	replaced bool // The next update releases holds on the previous layout.

	// landed is the key each pointer that is down landed on, or nil.
	landed map[PointerID]*keyState
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

const noPosition = ebiten.Key(-1)

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
	layers, err := layout.resolve()
	if err != nil {
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

	k := &Keyboard{
		pointers: o.pointers, input: o.input, write: o.write, updated: never,
		layers: layers, landed: make(map[PointerID]*keyState),
	}
	k.place()

	return k, nil
}

// SetLayout replaces the layout and immediately shows its first layer in the current
// rectangle. It keeps the keyboard's visibility, pointers and input. At the next Update,
// keys held on the old layout are released. Pointers already holding them must lift and
// land again to press a key. If the layout is invalid, nothing changes.
func (k *Keyboard) SetLayout(layout Layout) error {
	layers, err := layout.resolve()
	if err != nil {
		return err
	}

	k.layers, k.layer, k.shift = layers, 0, StateIdle
	k.replaced = true
	k.place()
	return nil
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
	for _, row := range k.layers[k.layer].rows {
		for i := range row {
			keys = append(keys, k.placed(&row[i]))
		}
	}
	return keys
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

	if k.replaced {
		k.cancelLandings()
		k.replaced = false
	}

	if !listening {
		// What it let go of was let go of while nobody was told: the game has seen
		// those keys go up already, and what comes now is no key pressed again.
		k.changes = k.changes[:0]
	}

	if target := k.land(listening); target != k.layer {
		k.cancelLandings()
		k.layer = target
		k.place()
	}
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

// cancelLandings releases keys from the departing layer. A pointer that stays down
// still belongs to its original landing and cannot press a key on the new layer.
func (k *Keyboard) cancelLandings() {
	for id, key := range k.landed {
		k.release(key)
		k.landed[id] = nil
	}
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
// no key until it lifts. One that lifted again within the tick lets go at once. All
// landings hit the layer that was visible, and the first switch to another layer wins.
func (k *Keyboard) land(listening bool) int {
	target := k.layer
	for _, pointer := range k.down {
		if _, known := k.landed[pointer.ID]; known {
			continue
		}

		key := k.landingKey(pointer, listening)
		if key != nil && key.target != noLayer {
			if target == k.layer {
				target = key.target
			}
			k.landed[pointer.ID] = nil
			continue
		}

		k.press(key)

		if pointer.Lifted {
			k.release(key)
			key = nil
		}

		k.landed[pointer.ID] = key
	}
	return target
}

func (k *Keyboard) landingKey(pointer Pointer, listening bool) *keyState {
	if k.hidden || (!listening && !pointer.Landed) {
		return nil
	}
	return k.keyAt(image.Pt(pointer.X, pointer.Y))
}

func (k *Keyboard) keyAt(at image.Point) *keyState {
	for _, row := range k.layers[k.layer].rows {
		for i := range row {
			if at.In(row[i].Rect) {
				return &row[i]
			}
		}
	}

	return nil
}

// press puts one more pointer on a key. The first one presses it.
func (k *Keyboard) press(key *keyState) {
	if key == nil {
		return
	}

	key.fingers++
	if key.fingers > 1 {
		return
	}

	key.State = StatePressed

	if position := key.position; position != noPosition {
		if k.held[position] == 0 {
			k.changed(position).pressed = true
			k.events = append(k.events, Event{Kind: KeyDown, Key: position})
		}

		k.held[position]++
	}

	k.activate(key)
}

// release takes one pointer off a key. The last one releases it.
func (k *Keyboard) release(key *keyState) {
	if key == nil {
		return
	}

	key.fingers--
	if key.fingers > 0 {
		return
	}

	key.State = StateIdle

	if position := key.position; position != noPosition {
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
	rows := k.layers[k.layer].rows
	heaviest := 0.0
	for _, row := range rows {
		heaviest = math.Max(heaviest, rowWeight(row))
	}

	unit := float64(k.rect.Dx()) / heaviest

	for i, row := range rows {
		top := k.rect.Min.Y + k.rect.Dy()*i/len(rows)
		bottom := k.rect.Min.Y + k.rect.Dy()*(i+1)/len(rows)
		left := float64(k.rect.Min.X) + (heaviest-rowWeight(row))*unit/2

		for j := range row {
			key := &row[j]
			right := left + key.weight()*unit

			key.Rect = image.Rect(int(math.Round(left)), top, int(math.Round(right)), bottom)

			left = right
		}
	}
}

func rowWeight(row []keyState) float64 {
	total := 0.0
	for _, key := range row {
		total += key.weight()
	}

	return total
}
