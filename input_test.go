package osk

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// desk is a physical keyboard a test types on, telling of its keys as Ebitengine does: how
// long each has been down, which went down and which went up in this tick, and what was typed
// in it. A key that goes down and up within one tick is pressed, just pressed and just
// released in it, with a duration of zero.
type desk struct {
	held     map[ebiten.Key]int
	pressed  map[ebiten.Key]bool
	released map[ebiten.Key]bool
	typed    []rune
}

func newDesk() *desk {
	return &desk{held: map[ebiten.Key]int{}, pressed: map[ebiten.Key]bool{}, released: map[ebiten.Key]bool{}}
}

func (d *desk) IsKeyPressed(key ebiten.Key) bool      { return d.held[key] > 0 || d.pressed[key] }
func (d *desk) IsKeyJustPressed(key ebiten.Key) bool  { return d.pressed[key] }
func (d *desk) IsKeyJustReleased(key ebiten.Key) bool { return d.released[key] }
func (d *desk) KeyPressDuration(key ebiten.Key) int   { return d.held[key] }
func (d *desk) AppendInputChars(runes []rune) []rune  { return append(runes, d.typed...) }

// tick moves the desk one tick on: held keys have been down one tick longer, and nothing has
// just gone down, gone up or been typed.
func (d *desk) tick() {
	for key := range d.held {
		d.held[key]++
	}

	d.pressed = map[ebiten.Key]bool{}
	d.released = map[ebiten.Key]bool{}
	d.typed = nil
}

func (d *desk) press(key ebiten.Key, typed rune) {
	d.held[key] = 1
	d.pressed[key] = true
	d.typed = append(d.typed, typed)
}

func (d *desk) release(key ebiten.Key) {
	delete(d.held, key)
	d.released[key] = true
}

// finger is one pointer a test puts on the screen and takes off.
type finger struct {
	down bool
	x, y int
}

func (f *finger) AppendPointers(pointers []Pointer) []Pointer {
	if !f.down {
		return pointers
	}

	return append(pointers, Pointer{ID: 1, X: f.x, Y: f.y})
}

// both is a physical keyboard and an on-screen Q key under one finger, feeding one input
// whose clock the test holds: now.
type both struct {
	physical *desk
	touch    *finger
	keyboard *Keyboard
	input    *Input
	now      int64
}

func newBoth(t *testing.T) *both {
	t.Helper()

	b := &both{physical: newDesk(), touch: &finger{x: 50, y: 50}, input: NewInput()}
	b.input.physical = b.physical
	b.input.tick = func() int64 { return b.now }

	layout := Layout{
		Positions: Positions{'q': ebiten.KeyQ},
		Layers:    []Layer{{Rows: [][]Key{{{Do: Type("q")}}}}},
	}

	keyboard, err := New(layout, WithPointers(b.touch), WithInput(b.input))
	if err != nil {
		t.Fatal(err)
	}

	keyboard.SetRect(image.Rect(0, 0, 100, 100))
	b.keyboard = keyboard

	return b
}

// tick starts the next tick: the game's clock and the physical keyboard move on. What the test
// does to the keys comes after it, and then the keyboard's update.
func (b *both) tick() {
	b.now++
	b.physical.tick()
}

// q is what the input says of Q in this tick.
type q struct {
	pressed, justPressed, justReleased bool
	duration                           int
}

func (b *both) q() q {
	return q{
		pressed:      b.input.IsKeyPressed(ebiten.KeyQ),
		justPressed:  b.input.IsKeyJustPressed(ebiten.KeyQ),
		justReleased: b.input.IsKeyJustReleased(ebiten.KeyQ),
		duration:     b.input.KeyPressDuration(ebiten.KeyQ),
	}
}

func TestAHoldThatPassesFromThePhysicalKeyToTheScreenIsOneHold(t *testing.T) {
	b := newBoth(t)

	b.physical.press(ebiten.KeyQ, 'q')
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, justPressed: true, duration: 1}); got != want {
		t.Errorf("a physical press alone: the input says %+v, want %+v", got, want)
	}

	b.tick()
	b.touch.down = true
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, duration: 2}); got != want {
		t.Errorf("the finger joins a key that is physically down: the input says %+v, want %+v", got, want)
	}

	b.tick()
	b.physical.release(ebiten.KeyQ)
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, duration: 3}); got != want {
		t.Errorf("the physical key goes up under the finger: the input says %+v, want %+v", got, want)
	}

	b.tick()
	b.touch.down = false
	b.keyboard.Update()

	if got, want := b.q(), (q{justReleased: true}); got != want {
		t.Errorf("the finger lifts last: the input says %+v, want %+v", got, want)
	}

	if got := b.input.AppendJustReleasedKeys(nil); len(got) != 1 || got[0] != ebiten.KeyQ {
		t.Errorf("the keys just released are %v, want [Q]", got)
	}

	b.tick()
	b.keyboard.Update()

	if got := b.q(); got != (q{}) {
		t.Errorf("the tick after the release the input still says %+v", got)
	}
}

func TestAHoldThatPassesFromTheScreenToThePhysicalKeyIsOneHold(t *testing.T) {
	b := newBoth(t)

	b.touch.down = true
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, justPressed: true, duration: 1}); got != want {
		t.Errorf("a touch alone: the input says %+v, want %+v", got, want)
	}

	b.tick()
	b.physical.press(ebiten.KeyQ, 'q')
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, duration: 2}); got != want {
		t.Errorf("the physical key joins a key the finger holds: the input says %+v, want %+v", got, want)
	}

	b.tick()
	b.touch.down = false
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, duration: 3}); got != want {
		t.Errorf("the finger lifts off a key that is physically down: the input says %+v, want %+v", got, want)
	}

	b.tick()
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, duration: 4}); got != want {
		t.Errorf("the physical key holds on alone: the input says %+v, want %+v", got, want)
	}

	// The finger comes back: still the same hold.
	b.tick()
	b.touch.down = true
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, duration: 5}); got != want {
		t.Errorf("the finger comes back to the held key: the input says %+v, want %+v", got, want)
	}

	b.tick()
	b.touch.down = false
	b.physical.release(ebiten.KeyQ)
	b.keyboard.Update()

	if got, want := b.q(), (q{justReleased: true}); got != want {
		t.Errorf("both let go in one tick: the input says %+v, want %+v", got, want)
	}

	// A new physical press is a new hold: nothing of the old one is left in its duration.
	b.tick()
	b.physical.press(ebiten.KeyQ, 'q')
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, justPressed: true, duration: 1}); got != want {
		t.Errorf("a new physical press after the hold: the input says %+v, want %+v", got, want)
	}
}

func TestAPhysicalKeyTappedBetweenTwoTicksKeepsBothItsEdges(t *testing.T) {
	b := newBoth(t)

	b.keyboard.Update()

	// Down and up before the game's next update, as a quick backspace is.
	b.tick()
	b.physical.press(ebiten.KeyQ, 'q')
	b.physical.release(ebiten.KeyQ)
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, justPressed: true, justReleased: true}); got != want {
		t.Errorf("a physical tap within one tick: the input says %+v, want %+v, as Ebitengine does", got, want)
	}

	b.tick()
	b.keyboard.Update()

	if got := b.q(); got != (q{}) {
		t.Errorf("the tick after the tap the input still says %+v", got)
	}
}

func TestTheCharactersOfBothKeyboardsArriveInTheirTick(t *testing.T) {
	b := newBoth(t)

	b.physical.press(ebiten.KeyA, 'a')
	b.touch.down = true
	b.keyboard.Update()

	if got := string(b.input.AppendInputChars([]rune("…"))); got != "…aq" {
		t.Errorf("the input gave %q, want what was there, then the physical a, then the q on the screen", got)
	}

	b.tick()
	b.keyboard.Update()

	if got := string(b.input.AppendInputChars(nil)); got != "" {
		t.Errorf("the tick after, the input still gave %q", got)
	}
}

func TestAKeyboardThatIsNoLongerUpdatedLetsGoOfItsKeys(t *testing.T) {
	b := newBoth(t)

	b.touch.down = true
	b.keyboard.Update()

	// The game drops the keyboard with the finger on it, in the tick after it typed, and
	// updates another in its place.
	other, err := New(Layout{Layers: []Layer{{Rows: [][]Key{{{Do: Type("w")}}}}}}, WithPointers(&finger{}), WithInput(b.input))
	if err != nil {
		t.Fatal(err)
	}

	b.tick()
	other.Update()

	if got, want := b.q(), (q{justReleased: true}); got != want {
		t.Errorf("the tick its keyboard is dropped, the input says of the key it held %+v, want %+v", got, want)
	}

	if got := string(b.input.AppendInputChars(nil)); got != "" {
		t.Errorf("the dropped keyboard's character arrived again: %q", got)
	}

	b.tick()
	other.Update()

	if got := b.q(); got != (q{}) {
		t.Errorf("two ticks after its keyboard was dropped, the input still says %+v", got)
	}

	// Updated again, the keyboard is heard again. Its finger never lifted: it holds the key.
	b.tick()
	b.keyboard.Update()
	other.Update()

	if got := b.q(); !got.pressed {
		t.Errorf("a keyboard that is updated again is not heard: the input says %+v", got)
	}
}

func TestAKeyEbitengineDoesNotKnowIsNeverPressed(t *testing.T) {
	b := newBoth(t)

	b.touch.down = true
	b.keyboard.Update()

	for _, key := range []ebiten.Key{-1, ebiten.KeyMax + 1} {
		if b.input.IsKeyPressed(key) || b.input.IsKeyJustPressed(key) || b.input.IsKeyJustReleased(key) || b.input.KeyPressDuration(key) != 0 {
			t.Errorf("the key %d, which Ebitengine does not have, is reported as pressed or released", key)
		}
	}
}

func TestAPositionEbitengineDoesNotKnowIsTextOnly(t *testing.T) {
	layout := Layout{
		Positions: Positions{'x': ebiten.KeyMax + 1, 'y': -1},
		Layers:    []Layer{{Rows: [][]Key{{{Do: Type("x")}, {Do: Type("y")}}}}},
	}

	touch := &finger{down: true, x: 10, y: 10}
	input := StillInput()

	keyboard, err := New(layout, WithPointers(touch), WithInput(input))
	if err != nil {
		t.Fatal(err)
	}

	keyboard.SetRect(image.Rect(0, 0, 100, 100))
	keyboard.Update()

	if got := input.AppendPressedKeys(nil); len(got) != 0 {
		t.Errorf("a position outside Ebitengine's keys pressed %v", got)
	}

	if got := string(input.AppendInputChars(nil)); got != "x" {
		t.Errorf("the key typed %q, want x", got)
	}
}
