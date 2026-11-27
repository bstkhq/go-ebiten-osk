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

func TestAHoldThatPassesToTheScreenInTheTickThePhysicalKeyGoesUpIsOneHold(t *testing.T) {
	b := newBoth(t)

	b.physical.press(ebiten.KeyQ, 'q')
	b.keyboard.Update()

	for range 5 {
		b.tick()
		b.keyboard.Update()
	}

	if got, want := b.q(), (q{pressed: true, duration: 6}); got != want {
		t.Fatalf("a physical key held for six ticks: the input says %+v, want %+v", got, want)
	}

	// Between two updates the finger lands and the physical key goes up: the two never
	// held the key in the same update, and the key was never up in one.
	b.tick()
	b.physical.release(ebiten.KeyQ)
	b.touch.down = true
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, duration: 7}); got != want {
		t.Errorf("the tick the hold passes to the screen: the input says %+v, want %+v", got, want)
	}

	// And back, the same way.
	b.tick()
	b.touch.down = false
	b.physical.press(ebiten.KeyQ, 'q')
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, duration: 8}); got != want {
		t.Errorf("the tick the hold passes back to the physical key: the input says %+v, want %+v", got, want)
	}
}

func TestAHoldThatPassesBetweenTwoKeyboardsOnTheScreenIsOneHold(t *testing.T) {
	b := newBoth(t)

	// A second keyboard with its own Q, under its own finger, feeding the same input.
	touch := &finger{x: 50, y: 50}

	other, err := New(Layout{
		Positions: Positions{'q': ebiten.KeyQ},
		Layers:    []Layer{{Rows: [][]Key{{{Do: Type("q")}}}}},
	}, WithPointers(touch), WithInput(b.input))
	if err != nil {
		t.Fatal(err)
	}

	other.SetRect(image.Rect(0, 0, 100, 100))

	update := func() {
		b.keyboard.Update()
		other.Update()
	}

	// The first holds Q on ticks one to three, the second on ticks three and four.
	b.touch.down = true
	update()

	b.tick()
	update()

	b.tick()
	touch.down = true
	update()

	if got, want := b.q(), (q{pressed: true, duration: 3}); got != want {
		t.Errorf("both keyboards hold the key: the input says %+v, want %+v", got, want)
	}

	if got := string(b.input.AppendInputChars(nil)); got != "q" {
		t.Errorf("the second keyboard's key typed %q, want its q", got)
	}

	b.tick()
	b.touch.down = false
	update()

	if got, want := b.q(), (q{pressed: true, duration: 4}); got != want {
		t.Errorf("the first keyboard let go and the second holds on: the input says %+v, want %+v", got, want)
	}

	b.tick()
	touch.down = false
	update()

	if got, want := b.q(), (q{justReleased: true}); got != want {
		t.Errorf("the last keyboard let go: the input says %+v, want %+v", got, want)
	}
}

func TestAKeyboardThatIsNoLongerUpdatedLetsGoOfItsKeysOnce(t *testing.T) {
	b := newBoth(t)

	b.touch.down = true
	b.keyboard.Update()

	// The game closes its only keyboard with the finger on it and goes on reading the input:
	// nothing is updated any more.
	b.tick()

	if got, want := b.q(), (q{justReleased: true}); got != want {
		t.Errorf("the tick after its keyboard's last update, the input says of the key it held %+v, want %+v", got, want)
	}

	if got := string(b.input.AppendInputChars(nil)); got != "" {
		t.Errorf("the dropped keyboard's character arrived again: %q", got)
	}

	for range 3 {
		b.tick()

		if got := b.q(); got != (q{}) {
			t.Fatalf("at tick %d, well after its keyboard was dropped, the input still says %+v", b.now, got)
		}
	}

	// A physical press afterwards is a press like any other.
	b.tick()
	b.physical.press(ebiten.KeyQ, 'q')

	if got, want := b.q(), (q{pressed: true, justPressed: true, duration: 1}); got != want {
		t.Errorf("a physical press after the keyboard was dropped: the input says %+v, want %+v", got, want)
	}
}

func TestAKeyboardThatIsUpdatedAgainPressesWhatItStillHolds(t *testing.T) {
	b := newBoth(t)

	b.touch.down = true
	b.keyboard.Update()

	b.tick()
	b.keyboard.Update()

	// The game skips the keyboard's update for a tick, with the finger still down, and sees
	// the key go up.
	b.tick()

	if got, want := b.q(), (q{justReleased: true}); got != want {
		t.Fatalf("the tick the keyboard is not updated: the input says %+v, want %+v", got, want)
	}

	// It updates it again: what the game sees is a new press, typed once, long ago.
	b.tick()
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, justPressed: true, duration: 1}); got != want {
		t.Errorf("the tick the keyboard is updated again: the input says %+v, want %+v", got, want)
	}

	if got := string(b.input.AppendInputChars(nil)); got != "" {
		t.Errorf("the key that was held all along typed %q again", got)
	}

	b.tick()
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, duration: 2}); got != want {
		t.Errorf("the tick after: the input says %+v, want %+v", got, want)
	}
}

func TestAKeyboardUpdatedTwiceInATickTypesOnce(t *testing.T) {
	b := newBoth(t)

	b.touch.down = true
	b.keyboard.Update()
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, justPressed: true, duration: 1}); got != want {
		t.Errorf("a keyboard updated twice in a tick: the input says %+v, want %+v", got, want)
	}

	if got := string(b.input.AppendInputChars(nil)); got != "q" {
		t.Errorf("a keyboard updated twice in a tick typed %q, want one q", got)
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
	input, _ := SteppedInput()

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

func TestAPhysicalKeyReleasedAndPressedAgainBetweenTwoTicksIsANewHold(t *testing.T) {
	b := newBoth(t)

	b.physical.press(ebiten.KeyQ, 'q')
	b.keyboard.Update()

	for range 6 {
		b.tick()
		b.keyboard.Update()
	}

	if got, want := b.q(), (q{pressed: true, duration: 7}); got != want {
		t.Fatalf("a physical key held for seven ticks: the input says %+v, want %+v", got, want)
	}

	// Up and down again before the next update: Ebitengine tells of a new press, one tick
	// old, and of the release before it. Nothing on the screen held the key meanwhile.
	b.tick()
	b.physical.release(ebiten.KeyQ)
	b.physical.press(ebiten.KeyQ, 'q')
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, justPressed: true, justReleased: true, duration: 1}); got != want {
		t.Errorf("a physical key released and pressed again within a tick: the input says %+v, want %+v, as Ebitengine does", got, want)
	}

	b.tick()
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, duration: 2}); got != want {
		t.Errorf("the tick after: the input says %+v, want %+v", got, want)
	}
}

func TestAPhysicalKeyPressedAgainUnderAFingerStaysOneHold(t *testing.T) {
	b := newBoth(t)

	b.physical.press(ebiten.KeyQ, 'q')
	b.touch.down = true
	b.keyboard.Update()

	b.tick()
	b.keyboard.Update()

	// The physical key goes up and down again while the finger holds the key on the screen:
	// to a game the key was never up.
	b.tick()
	b.physical.release(ebiten.KeyQ)
	b.physical.press(ebiten.KeyQ, 'q')
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, duration: 3}); got != want {
		t.Errorf("a physical key pressed again under a finger: the input says %+v, want %+v", got, want)
	}
}

func TestAHoldAGameWatchedWithNoKeyboardPassesToTheOneThatComes(t *testing.T) {
	b := newBoth(t)

	// The text field is closed: the game has no keyboard to update, and reads the input every
	// tick while the physical key is held.
	b.physical.press(ebiten.KeyQ, 'q')

	for held := 1; held <= 6; held++ {
		if got, want := b.q(), (q{pressed: true, justPressed: held == 1, duration: held}); got != want {
			t.Fatalf("tick %d of a physical hold with no keyboard: the input says %+v, want %+v", held, got, want)
		}

		b.tick()
	}

	// The field opens: its keyboard's first update comes with the finger on the key, in the
	// tick the physical key goes up.
	b.physical.release(ebiten.KeyQ)
	b.touch.down = true
	b.keyboard.Update()

	if got, want := b.q(), (q{pressed: true, duration: 7}); got != want {
		t.Errorf("the hold passes to a keyboard that was not there before: the input says %+v, want %+v", got, want)
	}
}

func TestAHoldNobodyWatchedPassesToTheScreenWithNoNewPress(t *testing.T) {
	b := newBoth(t)

	// The game neither reads the input nor updates a keyboard while the physical key is
	// held: nothing tells the input since when.
	b.physical.press(ebiten.KeyQ, 'q')

	for range 6 {
		b.tick()
	}

	b.physical.release(ebiten.KeyQ)
	b.touch.down = true
	b.keyboard.Update()

	// It was down when the tick before ended, so this is no new press: the hold is as old
	// as the input can vouch for.
	if got, want := b.q(), (q{pressed: true, duration: 2}); got != want {
		t.Errorf("a hold nobody watched passes to the screen: the input says %+v, want %+v", got, want)
	}
}

func TestAModifierNamedForEitherSideIsPressedWithOneOfItsKeysOnTheScreen(t *testing.T) {
	touch := &finger{down: true, x: 10, y: 10}
	input, _ := SteppedInput()

	keyboard, err := New(Layout{
		Positions: Positions{'⇧': ebiten.KeyShiftLeft},
		Layers:    []Layer{{Rows: [][]Key{{{Do: Type("⇧")}}}}},
	}, WithPointers(touch), WithInput(input))
	if err != nil {
		t.Fatal(err)
	}

	keyboard.SetRect(image.Rect(0, 0, 100, 100))

	if input.IsKeyPressed(ebiten.KeyShift) {
		t.Fatal("shift is pressed before anything touched it")
	}

	keyboard.Update()

	if !input.IsKeyPressed(ebiten.KeyShiftLeft) || !input.IsKeyPressed(ebiten.KeyShift) {
		t.Errorf("a key at the left shift's position: left shift pressed %t, shift pressed %t, want both",
			input.IsKeyPressed(ebiten.KeyShiftLeft), input.IsKeyPressed(ebiten.KeyShift))
	}

	if input.IsKeyPressed(ebiten.KeyShiftRight) || input.IsKeyPressed(ebiten.KeyControl) {
		t.Error("a key at the left shift's position pressed the right shift or control")
	}
}
