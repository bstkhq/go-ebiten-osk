package osk

import (
	"image"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// A Keyboard is a board of keys on the screen. It holds a layout and the rectangle it
// occupies, and every [Keyboard.Update] turns the pointers on its keys into key presses and
// typed text for its input. It draws nothing. Make one with [New].
type Keyboard struct {
	rect      image.Rectangle
	animation animation
	pointers  Pointers
	input     *Input
	write     InputWriter
	onAction  func(FieldAction)

	// updated is the tick of the last update. A keyboard listens for landings from one
	// update to the next: a pointer it finds down after a tick without an update came down
	// while it was not listening, and presses nothing.
	updated int64

	layers       []resolvedLayer
	layer        int
	shift        State
	releaseHolds bool // The next update releases holds after Hide or SetLayout.

	// landed records each pointer's ownership and the key it still holds, if any.
	landed map[PointerID]landing
	down   []Pointer

	repeat eraseRepeat

	// events is what the keyboard did in its last update, and changes what became of the
	// physical positions in it.
	events  []Event
	changes []change
	// writerEvents keeps the writer's borrowed copy separate from events.
	writerEvents []Event

	// held counts the keys pressed at each physical position: two keys of a layer may stand
	// for one position, which goes down with the first and up with the last.
	held [ebiten.KeyMax + 1]int
}

type landing struct {
	key    *keyState
	taken  bool
	lifted bool
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

	// Rect is the key's touch target on the screen, excluding any reserved row
	// gaps. A painter can inset it as its theme says.
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
	pointers  Pointers
	input     *Input
	repeat    eraseRepeat
	write     InputWriter
	onAction  func(FieldAction)
	animation time.Duration
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

// InputWriter receives a keyboard's held physical positions and a copy of its
// events after each update. The copy is valid only for the duration of the call.
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

	o := options{repeat: eraseRepeat{delay: 30, interval: 6}, animation: DefaultAnimationDuration}
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
		onAction: o.onAction,
		layers:   layers, landed: make(map[PointerID]landing), repeat: o.repeat,
		animation: animation{duration: o.animation, ticks: 1, step: 1, target: 1},
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
	k.releaseHolds = true
	k.place()
	return nil
}

// SetRect gives the keyboard the rectangle it occupies. The rows share its
// height, and keys and reserved gaps use the same weight scale. A row lighter
// than the heaviest is centered.
func (k *Keyboard) SetRect(rect image.Rectangle) {
	k.rect = rect
	k.place()
}

// Rect is the rectangle the keyboard occupies.
func (k *Keyboard) Rect() image.Rectangle {
	return k.rect
}

// Show brings the keyboard back, reversing an ongoing hide from its current
// position. The keyboard accepts key presses once it is fully shown.
func (k *Keyboard) Show() {
	k.animation.aim(true)
}

// Hide takes the keyboard away. Held keys are released at the start of the next
// Update, before that update advances the transition.
func (k *Keyboard) Hide() {
	k.animation.aim(false)
	k.releaseHolds = true
}

// Shown says whether the keyboard is showing or still on its way off screen.
func (k *Keyboard) Shown() bool {
	return k.animation.step != 0 || k.animation.target != 0
}

// AppendKeys appends the keys of the layer in view, as the keyboard holds them now, and
// returns the extended slice. It is what a painter draws.
func (k *Keyboard) AppendKeys(keys []PlacedKey) []PlacedKey {
	for _, row := range k.layers[k.layer].rows {
		for i := range row.keys {
			keys = append(keys, k.placed(&row.keys[i]))
		}
	}
	return keys
}

// AppendEvents appends what the keyboard did in its last update, in the order it happened,
// and returns the extended slice.
func (k *Keyboard) AppendEvents(events []Event) []Event {
	return append(events, k.events...)
}

// PointerTaken reports whether id belongs to the keyboard in the current tick. The
// keyboard takes a pointer that lands anywhere in its rectangle while shown, including
// blank space and during transitions, and keeps it through slides and the update
// that sees a same-tick tap. Hiding the keyboard after Update keeps its pointers taken
// for that tick. Once fully hidden, the next update clears ownership. Call
// PointerTaken after Update, before handling the pointer elsewhere in the game.
func (k *Keyboard) PointerTaken(id PointerID) bool {
	return k.updated == k.input.tick() && k.landed[id].taken
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

	stock, _ := k.pointers.(*screen)
	if stock != nil {
		stock.adoptMouse(k, listening)
	}
	k.lift(listening)

	if k.releaseHolds {
		k.cancelLandings()
		k.releaseHolds = false
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
	if stock != nil {
		stock.rememberTouches(k)
	}
	k.repeatErase()
	k.animation.advance()
	k.deliver()
	k.dispatchActions()
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
	k.writerEvents = append(k.writerEvents[:0], k.events...)
	k.write(held, k.writerEvents)
}

// cancelLandings releases keys from the departing layer. A pointer that stays down
// still belongs to its original landing and cannot press a key on the new layer.
func (k *Keyboard) cancelLandings() {
	for id, landed := range k.landed {
		k.release(landed.key)
		landed.key = nil
		k.landed[id] = landed
	}
}

// lift releases the keys of the pointers that are no longer down, or that lifted and landed
// again since the last update; and of all of them while the keyboard is hidden or was not
// listening.
func (k *Keyboard) lift(listening bool) {
	for id, landed := range k.landed {
		if !listening || !k.Shown() || landed.lifted || !stayedDown(k.down, id) {
			k.release(landed.key)
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

		landed := k.take(pointer, listening)
		key := landed.key
		if key != nil && key.target != noLayer {
			if target == k.layer {
				target = key.target
			}
			landed.key = nil
		} else {
			k.press(key)
			if landed.lifted {
				k.release(key)
				landed.key = nil
			}
		}

		k.landed[pointer.ID] = landed
	}
	return target
}

func (k *Keyboard) take(pointer Pointer, listening bool) landing {
	if !k.Shown() || (!listening && !pointer.Landed) {
		return landing{lifted: pointer.Lifted}
	}

	at := image.Pt(pointer.X, pointer.Y)
	if stock, ok := k.pointers.(*screen); ok && pointer.ID == Mouse {
		return stock.takeMouse(k, at, pointer.Lifted)
	}
	return k.takeAt(at, pointer.Lifted)
}

func (k *Keyboard) takeAt(at image.Point, lifted bool) landing {
	if !at.In(k.rect) {
		return landing{lifted: lifted}
	}
	landed := landing{taken: true, lifted: lifted}
	if k.animation.acceptsKeys() {
		landed.key = k.keyAt(at)
	}
	return landed
}

func (k *Keyboard) keyAt(at image.Point) *keyState {
	for _, row := range k.layers[k.layer].rows {
		for i := range row.keys {
			if at.In(row.keys[i].Rect) {
				return &row.keys[i]
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
	if key.Do.kind == erases {
		k.repeat.press(k.updated)
	}

	if position := key.position; position != noPosition {
		if k.held[position] == 0 {
			k.positionDown(position)
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
	if key.Do.kind == erases {
		k.repeat.held--
	}

	if position := key.position; position != noPosition {
		if k.held[position] == 1 {
			k.positionUp(position)
		}

		k.held[position]--
	}
}

func (k *Keyboard) positionDown(position ebiten.Key) {
	k.changed(position).pressed = true
	k.events = append(k.events, Event{Kind: KeyDown, Key: position})
}

func (k *Keyboard) positionUp(position ebiten.Key) {
	k.changed(position)
	k.events = append(k.events, Event{Kind: KeyUp, Key: position})
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
		heaviest = math.Max(heaviest, row.weight)
	}

	unit := float64(k.rect.Dx()) / heaviest

	for i, row := range rows {
		top := k.rect.Min.Y + k.rect.Dy()*i/len(rows)
		bottom := k.rect.Min.Y + k.rect.Dy()*(i+1)/len(rows)
		left := float64(k.rect.Min.X) + (heaviest-row.weight)*unit/2

		for j := range row.keys {
			key := &row.keys[j]
			left += key.leadingGap * unit
			right := left + key.weight()*unit

			key.Rect = image.Rect(int(math.Round(left)), top, int(math.Round(right)), bottom)

			left = right
		}
	}
}
