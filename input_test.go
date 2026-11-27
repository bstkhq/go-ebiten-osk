package osk

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// desk is a physical keyboard a test types on: how long each key has been down, which went up
// in this tick, and what was typed in it.
type desk struct {
	held     map[ebiten.Key]int
	released map[ebiten.Key]bool
	typed    []rune
}

func (d *desk) IsKeyPressed(key ebiten.Key) bool      { return d.held[key] > 0 }
func (d *desk) KeyPressDuration(key ebiten.Key) int   { return d.held[key] }
func (d *desk) IsKeyJustReleased(key ebiten.Key) bool { return d.released[key] }
func (d *desk) AppendInputChars(runes []rune) []rune  { return append(runes, d.typed...) }

// tick moves the desk one tick on: held keys have been down one tick longer, and nothing has
// just been released or typed.
func (d *desk) tick() {
	for key := range d.held {
		d.held[key]++
	}

	d.released = map[ebiten.Key]bool{}
	d.typed = nil
}

func (d *desk) press(key ebiten.Key, typed rune) {
	d.held[key] = 1
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

// both is a physical keyboard and an on-screen Q key under one finger, feeding one input.
func both(t *testing.T) (*desk, *finger, *Keyboard, *Input) {
	t.Helper()

	physical := &desk{held: map[ebiten.Key]int{}, released: map[ebiten.Key]bool{}}
	input := &Input{physical: physical}
	touch := &finger{x: 50, y: 50}

	layout := Layout{
		Positions: Positions{'q': ebiten.KeyQ},
		Layers:    []Layer{{Rows: [][]Key{{{Do: Type("q")}}}}},
	}

	keyboard, err := New(layout, WithPointers(touch), WithInput(input))
	if err != nil {
		t.Fatal(err)
	}

	keyboard.SetRect(image.Rect(0, 0, 100, 100))

	return physical, touch, keyboard, input
}

func TestAKeyIsPressedWhileEitherKeyboardHoldsIt(t *testing.T) {
	physical, touch, keyboard, input := both(t)

	physical.press(ebiten.KeyQ, 'q')
	keyboard.Update()

	if !input.IsKeyJustPressed(ebiten.KeyQ) || input.KeyPressDuration(ebiten.KeyQ) != 1 {
		t.Error("a physical press alone must be a press of the input")
	}

	// The finger joins a key that is already down: no new press.
	physical.tick()
	touch.down = true
	keyboard.Update()

	if !input.IsKeyPressed(ebiten.KeyQ) || input.IsKeyJustPressed(ebiten.KeyQ) {
		t.Error("touching a key that is physically down must not press it again")
	}

	if got := input.KeyPressDuration(ebiten.KeyQ); got != 2 {
		t.Errorf("the key has been down for two ticks, the duration is %d", got)
	}

	// The physical key goes up; the finger still holds it.
	physical.tick()
	physical.release(ebiten.KeyQ)
	keyboard.Update()

	if !input.IsKeyPressed(ebiten.KeyQ) || input.IsKeyJustReleased(ebiten.KeyQ) {
		t.Error("a key the finger still holds must stay pressed when the physical one goes up")
	}

	// The finger lifts: now it is released.
	physical.tick()
	touch.down = false
	keyboard.Update()

	if input.IsKeyPressed(ebiten.KeyQ) || !input.IsKeyJustReleased(ebiten.KeyQ) {
		t.Error("the key must be released when both have let go")
	}

	if got := input.AppendJustReleasedKeys(nil); len(got) != 1 || got[0] != ebiten.KeyQ {
		t.Errorf("the keys just released are %v, want [Q]", got)
	}
}

func TestAPhysicalReleaseEndsAPressTheFingerLeftFirst(t *testing.T) {
	physical, touch, keyboard, input := both(t)

	touch.down = true
	keyboard.Update()

	physical.tick()
	physical.press(ebiten.KeyQ, 'q')
	keyboard.Update()

	if input.IsKeyJustPressed(ebiten.KeyQ) {
		t.Error("a physical press of a key the finger holds must not press it again")
	}

	physical.tick()
	touch.down = false
	keyboard.Update()

	if !input.IsKeyPressed(ebiten.KeyQ) || input.IsKeyJustReleased(ebiten.KeyQ) {
		t.Error("a key that is physically down must stay pressed when the finger lifts")
	}

	physical.tick()
	physical.release(ebiten.KeyQ)
	keyboard.Update()

	if !input.IsKeyJustReleased(ebiten.KeyQ) {
		t.Error("the key must be released when the physical one goes up last")
	}
}

func TestTheCharactersOfBothKeyboardsArriveInTheirTick(t *testing.T) {
	physical, touch, keyboard, input := both(t)

	physical.press(ebiten.KeyA, 'a')
	touch.down = true
	keyboard.Update()

	if got := string(input.AppendInputChars([]rune("…"))); got != "…aq" {
		t.Errorf("the input gave %q, want what was there, then the physical a, then the q on the screen", got)
	}

	physical.tick()
	keyboard.Update()

	if got := string(input.AppendInputChars(nil)); got != "" {
		t.Errorf("the tick after, the input still gave %q", got)
	}
}

func TestAKeyEbitengineDoesNotKnowIsNeverPressed(t *testing.T) {
	_, touch, keyboard, input := both(t)

	touch.down = true
	keyboard.Update()

	for _, key := range []ebiten.Key{-1, ebiten.KeyMax + 1} {
		if input.IsKeyPressed(key) || input.IsKeyJustPressed(key) || input.IsKeyJustReleased(key) || input.KeyPressDuration(key) != 0 {
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
	input := NewInput()

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
