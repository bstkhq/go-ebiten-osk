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

	// tick is the game's clock, and now the tick the positions were last brought to: the
	// input starts each tick the first time it is fed or asked in it.
	tick func() int64
	now  int64

	// positions is what the input knows of each physical position.
	positions [ebiten.KeyMax + 1]position

	// typed is what the on-screen keyboards typed in the tick now.
	typed []rune

	// screen is Ebitengine's touches, which the input watches in every tick it starts, so
	// that a keyboard opened later can tell a finger that lands from one that was down.
	screen *fingers
}

// never is a tick that never comes: the stamp of what has not happened.
const never = math.MinInt64 / 2

// NewInput is an input that hears the physical keyboard, and the keyboards made [WithInput]
// it.
func NewInput() *Input {
	in := &Input{physical: hardware{}, tick: ebiten.Tick, now: never, screen: &touches}

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

// A position is what an input knows of one physical position in a tick. It belongs to the
// input, not to a keyboard: a key held by one keyboard and then by another, physical or on
// the screen, is one key held.
//
// The physical keyboard and the on-screen ones are two holders of the key, each with its own
// presses and releases. To a game the key goes down when a press of one comes while the other
// was not holding it, and up when a release of one leaves the other not holding it. Everything
// it is told comes from the facts below, whatever the order the keyboards were updated in.
type position struct {
	// As the tick before ended: whether the key was down, since which tick it had been, and
	// whether an on-screen keyboard was among those holding it.
	wasDown       bool
	since         int64
	screenWasDown bool

	// The physical key in this tick: the tick its press began, or never while it is up;
	// whether it went down and up in the tick, and whether it was down as the tick before
	// ended.
	physicalSince    int64
	physicalPressed  bool
	physicalReleased bool
	physicalWasDown  bool

	// The on-screen keyboards in this tick: how many hold the key after their update; how
	// many of those let go of it and pressed it again between two updates; and how many
	// pressed it and hold it no more, as a click begun and ended between two updates does.
	holders, again, taps int
}

// nothing is a position nothing ever held.
var nothing = position{since: never, physicalSince: never}

// start brings the position from the tick before to the tick now, in which the physical key
// has been down for held ticks. watched says that the input was fed or asked in the tick
// before: if it was not, nothing on the screen was held in it.
func (p *position) start(now int64, watched bool, held int, pressed, released bool) {
	down := watched && p.down()
	if down {
		p.since = p.sinceIn(now - 1)
	}

	p.wasDown = down
	p.screenWasDown = watched && p.holders > 0
	p.holders, p.again, p.taps = 0, 0, 0

	p.physicalSince = never
	if held > 0 {
		p.physicalSince = now - int64(held) + 1
	}

	p.physicalPressed, p.physicalReleased = pressed, released
	p.physicalWasDown = held > 1 || released && !pressed

	// A physical key that was down as the tick before ended, in a tick nothing told the
	// input of, was down all the same: the hold is as old as that, at least.
	if p.physicalWasDown && !p.wasDown {
		p.wasDown, p.since = true, now-1
	}
}

// down says whether a keyboard holds the key as this tick ends.
func (p *position) down() bool {
	return p.physicalSince != never || p.holders > 0
}

// tapped says whether the key was pressed and let go on the screen within this tick, with
// no keyboard on the screen holding it after.
func (p *position) tapped() bool {
	return p.taps > 0 && p.holders == 0
}

// screenPressed says whether the key went down on the screen in this tick, and screenReleased
// whether it went up there. A keyboard that lets go and presses again leaves the key up for a
// moment only if no other on the screen held it meanwhile.
func (p *position) screenPressed() bool {
	return p.pressedAgain() || p.tapped() || p.holders > 0 && !p.screenWasDown
}

func (p *position) screenReleased() bool {
	return p.pressedAgain() || p.tapped() || p.holders == 0 && p.screenWasDown
}

func (p *position) pressedAgain() bool {
	return p.again > 0 && p.again == p.holders
}

// justPressed says whether the key went down in this tick: a press of one holder while the
// other was not holding it as the tick before ended.
func (p *position) justPressed() bool {
	return p.physicalPressed && !p.screenWasDown || p.screenPressed() && !p.physicalWasDown
}

// justReleased says whether the key went up in this tick: a release of one holder that leaves
// the other not holding it.
func (p *position) justReleased() bool {
	return p.physicalReleased && p.holders == 0 || p.screenReleased() && p.physicalSince == never
}

// sinceIn is the first tick of the hold the key is in, in the tick now. It is asked only of a
// key that is down.
func (p *position) sinceIn(now int64) int64 {
	switch {
	case p.justPressed():
		return now
	case p.wasDown && p.physicalSince != never:
		return min(p.since, p.physicalSince)
	case p.wasDown:
		return p.since
	default:
		// Held on the physical keyboard since before the input was first asked.
		return p.physicalSince
	}
}

// begin starts the tick now, the first time the input is fed or asked in it: every position
// is brought to it, with the physical keyboard's state for the tick. It is what lets a hold
// pass from the physical key to the on-screen one and keep its age: once the physical key is
// up, Ebitengine has forgotten how long it was down.
func (in *Input) begin() int64 {
	now := in.tick()
	if now == in.now {
		return now
	}

	watched := in.now == now-1
	in.now = now
	in.typed = in.typed[:0]

	if in.screen != nil {
		in.screen.watch()
	}

	for i := range in.positions {
		key := ebiten.Key(i)
		in.positions[i].start(now, watched,
			in.physical.KeyPressDuration(key), in.physical.IsKeyJustPressed(key), in.physical.IsKeyJustReleased(key))
	}

	return now
}

// feed takes in an on-screen keyboard's update of this tick: the positions it holds, those
// that changed in the update, and what it typed.
func (in *Input) feed(held *[ebiten.KeyMax + 1]int, changes []change, events []Event) {
	in.begin()

	for key, fingers := range held {
		if fingers > 0 {
			in.positions[key].holders++
		}
	}

	for _, change := range changes {
		switch p := &in.positions[change.key]; {
		case !change.pressed:
		case held[change.key] == 0:
			p.taps++
		case change.before:
			p.again++
		}
	}

	for _, event := range events {
		if event.Kind == Text {
			in.typed = append(in.typed, []rune(event.Text)...)
		}
	}
}

// at is the position of the key in this tick, and the tick. A key Ebitengine does not have is
// a position nothing ever held.
func (in *Input) at(key ebiten.Key) (*position, int64) {
	now := in.begin()

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
// modifier named for either side is pressed while one of its two keys is, and a key pressed
// and let go between two ticks is pressed in the one tick.
func (in *Input) IsKeyPressed(key ebiten.Key) bool {
	if both, ok := sides[key]; ok {
		return in.IsKeyPressed(both[0]) || in.IsKeyPressed(both[1])
	}

	position, _ := in.at(key)

	return in.physical.IsKeyPressed(key) || position.holders > 0 || position.tapped()
}

// AppendInputChars appends the characters typed in this tick, as [ebiten.AppendInputChars]
// does, and returns the extended slice.
func (in *Input) AppendInputChars(runes []rune) []rune {
	in.begin()

	return append(in.physical.AppendInputChars(runes), in.typed...)
}

// IsKeyJustPressed says whether the key went down in this tick, as
// [inpututil.IsKeyJustPressed] does. A key one keyboard already held does not go down again
// when the other presses it.
func (in *Input) IsKeyJustPressed(key ebiten.Key) bool {
	position, _ := in.at(key)

	return position.justPressed()
}

// IsKeyJustReleased says whether the key went up in this tick, as
// [inpututil.IsKeyJustReleased] does. A key goes up when the last keyboard that held it lets
// go.
func (in *Input) IsKeyJustReleased(key ebiten.Key) bool {
	position, _ := in.at(key)

	return position.justReleased()
}

// KeyPressDuration is how many ticks the key has been pressed for, as
// [inpututil.KeyPressDuration] counts them: zero for a key pressed and let go between two
// ticks. A hold that passes from one keyboard to another without the key going up is one
// hold.
func (in *Input) KeyPressDuration(key ebiten.Key) int {
	position, now := in.at(key)

	if !position.down() {
		return 0
	}

	return int(now - position.sinceIn(now) + 1)
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
