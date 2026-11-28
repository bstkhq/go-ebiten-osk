package osk

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// A Pointer is one finger on the screen, or the mouse with its button down, and where it is,
// in the pixels the keyboard's rectangle is given in.
type Pointer struct {
	ID   PointerID
	X, Y int

	// Landed says that the pointer came down in this tick, even if one of the same
	// identifier was down in the tick before: it lifted and landed again between the two
	// updates, as the second of two quick clicks does. A keyboard that was updated in the
	// tick before takes for landed a pointer it did not see then, so a source that cannot
	// tell leaves it false.
	Landed bool

	// Lifted says that the pointer went up again within this tick: a click begun and ended
	// between two updates, which Ebitengine tells of as a button pressed for the one tick.
	// The key it landed on is pressed and let go in the same update.
	Lifted bool
}

// A PointerID tells one pointer from another for as long as it stays down.
type PointerID int

// Mouse is the mouse's pointer. A finger has the identifier Ebitengine gives its touch, which
// is never negative.
const Mouse PointerID = -1

// Pointers is where a keyboard learns which pointers are down in a tick. A keyboard reads
// Ebitengine's touches and mouse unless it is given another with [WithPointers]: a game that
// draws its keyboard on a surface it then scales or moves gives the pointers as that surface
// sees them.
type Pointers interface {
	// AppendPointers appends the pointers that are down now and returns the extended slice.
	AppendPointers(pointers []Pointer) []Pointer
}

// screen reads Ebitengine's own pointers. Each keyboard tracks echo matching against
// its own pointer ownership.
type screen struct {
	recent [3]touchSample
	mouse  mouseLanding
}

// mouseLanded says whether the left button went down in this tick, and mouseLifted whether it
// is up again already: Ebitengine tells of a click that began and ended within the tick as
// pressed, just pressed and just released, for no time at all.
func mouseLanded() bool {
	return inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
}

func mouseLifted() bool {
	return inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) &&
		inpututil.MouseButtonPressDuration(ebiten.MouseButtonLeft) == 0
}

func (*screen) AppendPointers(pointers []Pointer) []Pointer {
	touches.watch()

	// On a simultaneous landing the touch takes ownership before the mouse can match it.
	for _, touch := range touches.down {
		pointers = append(pointers, Pointer{
			ID: PointerID(touch.id), X: touch.at.X, Y: touch.at.Y, Landed: touches.landed(touch),
		})
	}

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		pointers = append(pointers, Pointer{ID: Mouse, X: x, Y: y, Landed: mouseLanded(), Lifted: mouseLifted()})
	}

	return pointers
}

// touches is Ebitengine's touches from one tick to the next. Ebitengine tells which fingers
// are down, not which have just come down: a keyboard that opens under fingers has to tell the
// one that lands in that very tick, which types, from one that was down already, which must
// not. So the touches are watched whenever an input is fed or asked, keyboard or no keyboard.
//
// They are the game's, like Ebitengine's own state: watched from the game's Update alone.
var touches fingers

type fingers struct {
	// at is the tick last watched, and seen whether any was.
	at   int64
	seen bool

	// before contains the preceding tick's contacts when it was watched, or none.
	ids          []ebiten.TouchID
	down, before []contact
	known        bool
}

type contact struct {
	id ebiten.TouchID
	at image.Point
}

// watch takes in the touches of this tick, the first time it is asked in it.
func (f *fingers) watch() {
	now := ebiten.Tick()
	if f.seen && now == f.at {
		return
	}

	f.known = f.seen && now == f.at+1
	f.before, f.down = f.down, f.before[:0]
	if !f.known {
		f.before = f.before[:0]
	}
	f.ids = ebiten.AppendTouchIDs(f.ids[:0])
	for _, id := range f.ids {
		x, y := ebiten.TouchPosition(id)
		f.down = append(f.down, contact{id: id, at: image.Pt(x, y)})
	}
	f.at, f.seen = now, true
}

const touchJump = 48

// landed says whether the touch came down in this tick. Where the tick before was not
// watched, it is what inpututil makes of the touch.
func (f *fingers) landed(touch contact) bool {
	if !f.known {
		return inpututil.TouchPressDuration(touch.id) == 1
	}
	for _, previous := range f.before {
		if previous.id == touch.id {
			return distance(previous.at, touch.at) > touchJump
		}
	}

	return true
}

func distance(a, b image.Point) float64 {
	return math.Hypot(float64(a.X)-float64(b.X), float64(a.Y)-float64(b.Y))
}

const mouseEchoRadius = 12

func withinMouseEcho(a, b image.Point) bool {
	return distance(a, b) <= mouseEchoRadius
}

type mouseLanding struct {
	at       image.Point
	tick     int64
	unpaired bool
}

type touchSample struct {
	tick     int64
	contacts []touchEcho
}

type touchEcho struct {
	at    image.Point
	taken bool
}

func (s *screen) rememberTouches(k *Keyboard) {
	sample := s.recent[len(s.recent)-1]
	sample.tick = k.updated
	sample.contacts = sample.contacts[:0]
	for _, touch := range touches.down {
		sample.contacts = append(sample.contacts, touchEcho{at: touch.at, taken: k.landed[PointerID(touch.id)].taken})
	}
	copy(s.recent[1:], s.recent[:len(s.recent)-1])
	s.recent[0] = sample
}

func (s *screen) matchTouch(k *Keyboard, at image.Point) (taken, matched bool) {
	for _, touch := range touches.down {
		if withinMouseEcho(at, touch.at) {
			return k.landed[PointerID(touch.id)].taken, true
		}
	}
	for _, sample := range s.recent {
		if k.updated-sample.tick > int64(len(s.recent)) {
			break
		}
		for _, touch := range sample.contacts {
			if withinMouseEcho(at, touch.at) {
				return touch.taken, true
			}
		}
	}
	return false, false
}

func (s *screen) takeMouse(k *Keyboard, at image.Point, lifted bool) landing {
	s.mouse.unpaired = false
	if taken, matched := s.matchTouch(k, at); matched {
		return landing{taken: taken || at.In(k.rect), lifted: lifted}
	}

	l := k.takeAt(at, lifted)
	s.mouse = mouseLanding{at: at, tick: k.updated, unpaired: l.taken}
	return l
}

// adoptMouse makes a touch reported one tick after its mouse echo carry the existing
// landing. The first report chose the key and already typed it. The finger now controls
// its release, even if the raw mouse button stays down after the finger lifts.
func (s *screen) adoptMouse(k *Keyboard, listening bool) {
	if !listening || k.hidden || !s.mouse.unpaired || s.mouse.tick != k.updated-1 {
		return
	}

	i := newTouchNear(k.down, s.mouse.at)
	if i < 0 {
		return
	}

	touch := &k.down[i]
	if prior, found := k.landed[touch.ID]; found {
		k.release(prior.key)
	}
	k.landed[touch.ID] = landing{key: k.landed[Mouse].key, taken: true}
	k.landed[Mouse] = landing{taken: true}
	touch.Landed = false // The landing already exists and continues through this sample.
}

func newTouchNear(down []Pointer, at image.Point) int {
	for i, pointer := range down {
		if pointer.ID != Mouse && pointer.Landed && withinMouseEcho(at, image.Pt(pointer.X, pointer.Y)) {
			return i
		}
	}
	return -1
}
