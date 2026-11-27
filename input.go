package osk

import (
	"math"

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
// feeds unless it is made [WithInput] another. Make one with [NewInput]: the zero value is not
// ready for use.
type Input struct {
	physical physical

	// tick is the game's clock. Everything the input keeps is stamped with it, so what was
	// fed in another tick is never taken for this one's.
	tick func() int64

	// positions is what the input knows of each physical position, and observed the tick
	// in which it last took the physical keyboard's holds into them.
	positions [ebiten.KeyMax + 1]position
	observed  int64

	// typed is what the on-screen keyboards typed in the tick typedAt.
	typed   []rune
	typedAt int64
}

// never is a tick that never comes: the stamp of what has not happened yet.
const never = math.MinInt64 / 2

// NewInput is an input that hears the physical keyboard, and the keyboards made [WithInput]
// it.
func NewInput() *Input {
	in := &Input{physical: hardware{}, tick: ebiten.Tick, observed: never, typedAt: never}

	for i := range in.positions {
		in.positions[i] = nothing
	}

	return in
}

var defaultInput = NewInput()

// physical is the hardware keyboard, as Ebitengine and inpututil tell of it. A key that goes
// down and up between two ticks is pressed, just pressed and just released in the one tick,
// with a duration of zero. One that goes up and down again is the same, with a duration of
// one.
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

// A position is what an input knows of one physical position. It belongs to the input, not to
// a keyboard: a key held by one keyboard and then by another, physical or on the screen, is
// one key held.
//
// The physical keyboard and the on-screen ones are two holders of the key, each with its own
// presses and releases. To a game the key goes down when a press of one comes while the other
// was not holding it, and up when a release of one leaves the other not holding it.
type position struct {
	// since and last are the first and the last tick of the hold the key is in, or was last
	// in: the latest run of ticks in which it was down without a break, whoever held it.
	since, last int64
	// physical is the tick in which the physical press last taken into that hold began.
	physical int64

	// screenSince and screenLast are the first and the last tick of the latest unbroken run
	// of ticks in which an on-screen keyboard held the key, and screenBy the keyboard that
	// last told of holding it.
	screenSince, screenLast int64
	screenBy                *Keyboard
	// screenAgain is the last tick in which that keyboard let go of the key and pressed it
	// again between two updates, with no other on the screen holding it: a release and a
	// press that leave the run unbroken.
	screenAgain int64
}

// nothing is a position nothing ever held.
var nothing = position{since: never, last: never, physical: never, screenSince: never, screenLast: never, screenAgain: never}

// screenDown says whether an on-screen keyboard holds the key in the tick now, and
// screenWasDown whether one held it when the tick before ended.
func (p *position) screenDown(now int64) bool {
	return p.screenLast == now
}

func (p *position) screenWasDown(now int64) bool {
	return p.screenLast == now-1 || p.screenLast == now && p.screenSince < now
}

// screenPressed says whether the key went down on the screen in the tick now, and
// screenReleased whether it went up there.
func (p *position) screenPressed(now int64) bool {
	return p.screenAgain == now || p.screenDown(now) && !p.screenWasDown(now)
}

func (p *position) screenReleased(now int64) bool {
	return p.screenAgain == now || p.screenWasDown(now) && !p.screenDown(now)
}

// physicalDown notes that the physical key is down in the tick now, in a press that began in
// the tick from. Two physical presses are two holds, as Ebitengine tells them, unless an
// on-screen keyboard held the key when the second came: then the key was never up.
func (p *position) physicalDown(now, from int64) {
	if p.physical != from {
		if p.screenLast >= from-1 {
			p.since = min(p.since, from)
		} else {
			p.since = from
		}

		p.physical = from
	}

	p.last = max(p.last, now)
}

// feed takes in an on-screen keyboard's update of this tick: the positions it holds, and what
// it did, from which come the presses of positions and the typed text.
func (in *Input) feed(keyboard *Keyboard, held *[ebiten.KeyMax + 1]int, events []Event) {
	now := in.tick()
	in.observe(now)

	if in.typedAt != now {
		in.typed, in.typedAt = in.typed[:0], now
	}

	for _, event := range events {
		switch event.Kind {
		case KeyDown:
			in.positions[event.Key].pressedAgain(keyboard, now)
		case Text:
			in.typed = append(in.typed, []rune(event.Text)...)
		}
	}

	for key, fingers := range held {
		if fingers > 0 {
			in.hold(ebiten.Key(key), keyboard, now)
		}
	}
}

// pressedAgain notes a press of the position by a keyboard that already held it when the tick
// before ended, with no other telling of it since: the keyboard let go and pressed again
// between its two updates, as the second of two quick clicks on a key does.
func (p *position) pressedAgain(keyboard *Keyboard, now int64) {
	if p.screenBy == keyboard && p.screenLast == now-1 {
		p.screenAgain = now
	}
}

// hold notes that the keyboard holds the position in this tick. A press on the screen begins a
// new hold unless the physical key was down when the tick before ended: then the hold passes
// to the screen, and goes on.
func (in *Input) hold(key ebiten.Key, keyboard *Keyboard, now int64) {
	p := &in.positions[key]
	physicalWasDown := in.physicalWasDown(key)

	// A physical key that was down when the tick before ended, and that nothing told the
	// input of in that tick, was down all the same: the hold is as old as that, at least.
	if physicalWasDown && p.last < now-1 {
		p.since, p.last = now-1, now-1
	}

	if p.screenLast < now-1 {
		p.screenSince = now
	}

	p.screenLast, p.screenBy = now, keyboard

	if p.screenPressed(now) && !physicalWasDown {
		p.since = now
	}

	p.last = now
}

// observe takes the physical keyboard's holds of this tick into the positions, once a tick,
// the first time the input is fed or asked in it. It is what lets a hold pass from the
// physical key to the on-screen one and keep its age: once the physical key is up, Ebitengine
// has forgotten how long it was down.
func (in *Input) observe(now int64) {
	if in.observed == now {
		return
	}

	in.observed = now

	for key := range in.positions {
		if held := in.physical.KeyPressDuration(ebiten.Key(key)); held > 0 {
			in.positions[key].physicalDown(now, now-int64(held)+1)
		}
	}
}

// at is the position of the key in this tick, and the tick. A key Ebitengine does not have is
// a position nothing ever held.
func (in *Input) at(key ebiten.Key) (*position, int64) {
	now := in.tick()
	in.observe(now)

	if key < 0 || key > ebiten.KeyMax {
		return &nothing, now
	}

	return &in.positions[key], now
}

// sides are the two keys behind each of the names Ebitengine has for a modifier on either
// side of the keyboard.
var sides = map[ebiten.Key][2]ebiten.Key{
	ebiten.KeyAlt:     {ebiten.KeyAltLeft, ebiten.KeyAltRight},
	ebiten.KeyControl: {ebiten.KeyControlLeft, ebiten.KeyControlRight},
	ebiten.KeyShift:   {ebiten.KeyShiftLeft, ebiten.KeyShiftRight},
	ebiten.KeyMeta:    {ebiten.KeyMetaLeft, ebiten.KeyMetaRight},
}

// IsKeyPressed says whether the key is pressed, as [ebiten.IsKeyPressed] does. As there, a
// modifier named for either side is pressed while one of its two keys is.
func (in *Input) IsKeyPressed(key ebiten.Key) bool {
	if both, ok := sides[key]; ok {
		return in.IsKeyPressed(both[0]) || in.IsKeyPressed(both[1])
	}

	position, now := in.at(key)

	return in.physical.IsKeyPressed(key) || position.screenDown(now)
}

// AppendInputChars appends the characters typed in this tick, as [ebiten.AppendInputChars]
// does, and returns the extended slice.
func (in *Input) AppendInputChars(runes []rune) []rune {
	now := in.tick()
	in.observe(now)

	runes = in.physical.AppendInputChars(runes)

	if in.typedAt == now {
		runes = append(runes, in.typed...)
	}

	return runes
}

// IsKeyJustPressed says whether the key went down in this tick, as
// [inpututil.IsKeyJustPressed] does. A key one keyboard already held does not go down again
// when the other presses it.
func (in *Input) IsKeyJustPressed(key ebiten.Key) bool {
	position, now := in.at(key)

	return in.physical.IsKeyJustPressed(key) && !position.screenWasDown(now) ||
		position.screenPressed(now) && !in.physicalWasDown(key)
}

// IsKeyJustReleased says whether the key went up in this tick, as
// [inpututil.IsKeyJustReleased] does. A key goes up when the last keyboard that held it lets
// go.
func (in *Input) IsKeyJustReleased(key ebiten.Key) bool {
	position, now := in.at(key)

	return in.physical.IsKeyJustReleased(key) && !position.screenDown(now) ||
		position.screenReleased(now) && in.physical.KeyPressDuration(key) == 0
}

// KeyPressDuration is how many ticks the key has been pressed for, as
// [inpututil.KeyPressDuration] counts them. A hold that passes from one keyboard to another
// without the key going up is one hold.
func (in *Input) KeyPressDuration(key ebiten.Key) int {
	position, now := in.at(key)

	if position.last != now {
		return 0
	}

	return int(now - position.since + 1)
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
