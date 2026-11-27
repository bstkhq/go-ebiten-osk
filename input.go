package osk

import (
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// An Input is the input copy as a value: the keyboard-reading functions of Ebitengine and of
// inpututil, answering for the physical keyboard and for the on-screen keyboards that feed it,
// together. A key is pressed while either holds it and released when both have let go, and
// the characters of both arrive in the tick they were typed.
//
// It answers for an on-screen keyboard in the ticks that keyboard was updated in, so a game
// updates its keyboard before it reads. A keyboard that is no longer updated lets go of every
// key it held.
//
// The package's functions of the same names answer from a default Input that every keyboard
// feeds unless it is made [WithInput] another.
type Input struct {
	physical physical

	// keyboards are the on-screen keyboards that fed the input in this tick or in the one
	// before. tick is the game's clock, by which the two are told apart.
	keyboards []*Keyboard
	tick      func() int64
}

// NewInput is an input that hears the physical keyboard, and the keyboards made [WithInput]
// it.
func NewInput() *Input {
	return &Input{physical: hardware{}, tick: ebiten.Tick}
}

var defaultInput = NewInput()

// physical is the hardware keyboard, as Ebitengine and inpututil tell of it. A key that goes
// down and up between two ticks is pressed, just pressed and just released in the one tick,
// with a duration of zero.
type physical interface {
	IsKeyPressed(key ebiten.Key) bool
	IsKeyJustPressed(key ebiten.Key) bool
	IsKeyJustReleased(key ebiten.Key) bool
	KeyPressDuration(key ebiten.Key) int
	AppendInputChars(runes []rune) []rune
}

type hardware struct{}

func (hardware) IsKeyPressed(key ebiten.Key) bool      { return ebiten.IsKeyPressed(key) }
func (hardware) IsKeyJustPressed(key ebiten.Key) bool  { return inpututil.IsKeyJustPressed(key) }
func (hardware) IsKeyJustReleased(key ebiten.Key) bool { return inpututil.IsKeyJustReleased(key) }
func (hardware) KeyPressDuration(key ebiten.Key) int   { return inpututil.KeyPressDuration(key) }
func (hardware) AppendInputChars(runes []rune) []rune  { return ebiten.AppendInputChars(runes) }

// IsKeyPressed says whether the key is pressed, as [ebiten.IsKeyPressed] does.
func (in *Input) IsKeyPressed(key ebiten.Key) bool {
	return in.physical.IsKeyPressed(key) || in.onScreen(key).down
}

// AppendInputChars appends the characters typed in this tick, as [ebiten.AppendInputChars]
// does, and returns the extended slice.
func (in *Input) AppendInputChars(runes []rune) []rune {
	runes = in.physical.AppendInputChars(runes)

	for _, keyboard := range in.keyboards {
		if in.updatedNow(keyboard) {
			runes = append(runes, keyboard.typed...)
		}
	}

	return runes
}

// IsKeyJustPressed says whether the key went down in this tick, as
// [inpututil.IsKeyJustPressed] does. A key one keyboard already held does not go down again
// when the other presses it.
func (in *Input) IsKeyJustPressed(key ebiten.Key) bool {
	screen := in.onScreen(key)

	if in.physical.IsKeyJustPressed(key) {
		return !screen.wasDown
	}

	return screen.down && !screen.wasDown && !in.physicalWasDown(key)
}

// IsKeyJustReleased says whether the key went up in this tick, as
// [inpututil.IsKeyJustReleased] does. A key goes up when the last keyboard that held it lets
// go.
func (in *Input) IsKeyJustReleased(key ebiten.Key) bool {
	screen := in.onScreen(key)

	if in.physical.IsKeyJustReleased(key) {
		return !screen.down
	}

	return screen.wasDown && !screen.down && in.physical.KeyPressDuration(key) == 0
}

// KeyPressDuration is how many ticks the key has been pressed for, as
// [inpututil.KeyPressDuration] counts them. A hold that passes from one keyboard to the other
// without the key going up is one hold.
func (in *Input) KeyPressDuration(key ebiten.Key) int {
	screen := in.onScreen(key)

	held := in.physical.KeyPressDuration(key)
	if held > 0 {
		held += screen.older
	}

	return max(held, screen.ticks)
}

// AppendPressedKeys appends the keys that are pressed, as [inpututil.AppendPressedKeys] does,
// and returns the extended slice.
func (in *Input) AppendPressedKeys(keys []ebiten.Key) []ebiten.Key {
	return appendKeys(keys, in.IsKeyPressed)
}

// AppendJustPressedKeys appends the keys that went down in this tick, as
// [inpututil.AppendJustPressedKeys] does, and returns the extended slice.
func (in *Input) AppendJustPressedKeys(keys []ebiten.Key) []ebiten.Key {
	return appendKeys(keys, in.IsKeyJustPressed)
}

// AppendJustReleasedKeys appends the keys that went up in this tick, as
// [inpututil.AppendJustReleasedKeys] does, and returns the extended slice.
func (in *Input) AppendJustReleasedKeys(keys []ebiten.Key) []ebiten.Key {
	return appendKeys(keys, in.IsKeyJustReleased)
}

func appendKeys(keys []ebiten.Key, is func(ebiten.Key) bool) []ebiten.Key {
	for key := ebiten.Key(0); key <= ebiten.KeyMax; key++ {
		if is(key) {
			keys = append(keys, key)
		}
	}

	return keys
}

// physicalWasDown says whether the physical key was down when the tick before this one ended:
// it has been down for more than this tick, or it went up in this one without going down in
// it.
func (in *Input) physicalWasDown(key ebiten.Key) bool {
	return in.physical.KeyPressDuration(key) > 1 ||
		in.physical.IsKeyJustReleased(key) && !in.physical.IsKeyJustPressed(key)
}

// onScreenKey is what the on-screen keyboards hold of a physical position in this tick.
type onScreenKey struct {
	// down says whether one of them holds it now, and wasDown whether one held it when the
	// tick before ended.
	down, wasDown bool
	// ticks is how long the hold they are part of has lasted, and older by how much a hold
	// they left to the physical key is older than that key's own press.
	ticks, older int
}

// onScreen gathers the position from the keyboards updated in this tick. One updated in the
// tick before and not in this one was dropped, or hidden from the game's updates: what it held
// then has gone up.
func (in *Input) onScreen(key ebiten.Key) onScreenKey {
	var screen onScreenKey

	if key < 0 || key > ebiten.KeyMax {
		return screen
	}

	for _, keyboard := range in.keyboards {
		presses := &keyboard.presses

		if !in.updatedNow(keyboard) {
			screen.wasDown = screen.wasDown || presses.ticks[key] > 0

			continue
		}

		screen.down = screen.down || presses.ticks[key] > 0
		screen.wasDown = screen.wasDown || presses.was[key]
		screen.ticks = max(screen.ticks, presses.ticks[key])
		screen.older = max(screen.older, presses.older[key])
	}

	return screen
}

// hear takes the keyboard's update of this tick in, and forgets the keyboards whose last
// update is older than the tick before.
func (in *Input) hear(keyboard *Keyboard) {
	now := in.tick()
	keyboard.updated = now

	in.keyboards = slices.DeleteFunc(in.keyboards, func(other *Keyboard) bool {
		return other == keyboard || now-other.updated > 1
	})
	in.keyboards = append(in.keyboards, keyboard)
}

func (in *Input) updatedNow(keyboard *Keyboard) bool {
	return keyboard.updated == in.tick()
}

// IsKeyPressed says whether the key is pressed on the physical keyboard or on the screen. It
// has the signature of [ebiten.IsKeyPressed].
func IsKeyPressed(key ebiten.Key) bool {
	return defaultInput.IsKeyPressed(key)
}

// AppendInputChars appends the characters typed in this tick on the physical keyboard and on
// the screen, and returns the extended slice. It has the signature of
// [ebiten.AppendInputChars].
func AppendInputChars(runes []rune) []rune {
	return defaultInput.AppendInputChars(runes)
}

// IsKeyJustPressed says whether the key went down in this tick. It has the signature of
// [inpututil.IsKeyJustPressed].
func IsKeyJustPressed(key ebiten.Key) bool {
	return defaultInput.IsKeyJustPressed(key)
}

// IsKeyJustReleased says whether the key went up in this tick. It has the signature of
// [inpututil.IsKeyJustReleased].
func IsKeyJustReleased(key ebiten.Key) bool {
	return defaultInput.IsKeyJustReleased(key)
}

// KeyPressDuration is how many ticks the key has been pressed for. It has the signature of
// [inpututil.KeyPressDuration].
func KeyPressDuration(key ebiten.Key) int {
	return defaultInput.KeyPressDuration(key)
}

// AppendPressedKeys appends the keys that are pressed and returns the extended slice. It has
// the signature of [inpututil.AppendPressedKeys].
func AppendPressedKeys(keys []ebiten.Key) []ebiten.Key {
	return defaultInput.AppendPressedKeys(keys)
}

// AppendJustPressedKeys appends the keys that went down in this tick and returns the extended
// slice. It has the signature of [inpututil.AppendJustPressedKeys].
func AppendJustPressedKeys(keys []ebiten.Key) []ebiten.Key {
	return defaultInput.AppendJustPressedKeys(keys)
}

// AppendJustReleasedKeys appends the keys that went up in this tick and returns the extended
// slice. It has the signature of [inpututil.AppendJustReleasedKeys].
func AppendJustReleasedKeys(keys []ebiten.Key) []ebiten.Key {
	return defaultInput.AppendJustReleasedKeys(keys)
}
