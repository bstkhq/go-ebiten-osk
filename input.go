package osk

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// An Input is the input copy as a value: the keyboard-reading functions of Ebitengine and of
// inpututil, answering for the physical keyboard and for the on-screen keyboards that feed it,
// together. A key is pressed while either holds it and released when both have let go, and
// the characters of both arrive in the tick they were typed.
//
// The package's functions of the same names answer from a default Input that every keyboard
// feeds unless it is made [WithInput] another.
type Input struct {
	physical  physical
	keyboards []*Keyboard
}

// NewInput is an input that hears the physical keyboard, and the keyboards made [WithInput]
// it.
func NewInput() *Input {
	return &Input{physical: hardware{}}
}

var defaultInput = NewInput()

// physical is the hardware keyboard, as Ebitengine and inpututil tell of it.
type physical interface {
	IsKeyPressed(key ebiten.Key) bool
	KeyPressDuration(key ebiten.Key) int
	IsKeyJustReleased(key ebiten.Key) bool
	AppendInputChars(runes []rune) []rune
}

type hardware struct{}

func (hardware) IsKeyPressed(key ebiten.Key) bool      { return ebiten.IsKeyPressed(key) }
func (hardware) KeyPressDuration(key ebiten.Key) int   { return inpututil.KeyPressDuration(key) }
func (hardware) IsKeyJustReleased(key ebiten.Key) bool { return inpututil.IsKeyJustReleased(key) }
func (hardware) AppendInputChars(runes []rune) []rune  { return ebiten.AppendInputChars(runes) }

// IsKeyPressed says whether the key is pressed, as [ebiten.IsKeyPressed] does.
func (in *Input) IsKeyPressed(key ebiten.Key) bool {
	return in.physical.IsKeyPressed(key) || in.onScreenTicks(key) > 0
}

// AppendInputChars appends the characters typed in this tick, as [ebiten.AppendInputChars]
// does, and returns the extended slice.
func (in *Input) AppendInputChars(runes []rune) []rune {
	runes = in.physical.AppendInputChars(runes)

	for _, keyboard := range in.keyboards {
		runes = append(runes, keyboard.typed...)
	}

	return runes
}

// IsKeyJustPressed says whether the key went down in this tick, as
// [inpututil.IsKeyJustPressed] does.
func (in *Input) IsKeyJustPressed(key ebiten.Key) bool {
	return in.IsKeyPressed(key) && !in.wasPressed(key)
}

// IsKeyJustReleased says whether the key went up in this tick, as
// [inpututil.IsKeyJustReleased] does.
func (in *Input) IsKeyJustReleased(key ebiten.Key) bool {
	return !in.IsKeyPressed(key) && in.wasPressed(key)
}

// KeyPressDuration is how many ticks the key has been pressed for, as
// [inpututil.KeyPressDuration] counts them.
func (in *Input) KeyPressDuration(key ebiten.Key) int {
	return max(in.physical.KeyPressDuration(key), in.onScreenTicks(key))
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

// wasPressed says whether the key was pressed, by either keyboard, in the tick before this
// one.
func (in *Input) wasPressed(key ebiten.Key) bool {
	if in.physical.KeyPressDuration(key) > 1 || in.physical.IsKeyJustReleased(key) {
		return true
	}

	if !known(key) {
		return false
	}

	for _, keyboard := range in.keyboards {
		if keyboard.presses.was[key] {
			return true
		}
	}

	return false
}

// onScreenTicks is how many ticks the on-screen keyboards have held the key for.
func (in *Input) onScreenTicks(key ebiten.Key) int {
	if !known(key) {
		return 0
	}

	ticks := 0
	for _, keyboard := range in.keyboards {
		ticks = max(ticks, keyboard.presses.ticks[key])
	}

	return ticks
}

func known(key ebiten.Key) bool {
	return key >= 0 && key <= ebiten.KeyMax
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
