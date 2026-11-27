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

// screen is Ebitengine's own pointers: the mouse while its left button is down, then the
// touches.
type screen struct{}

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

func (screen) AppendPointers(pointers []Pointer) []Pointer {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		pointers = append(pointers, Pointer{ID: Mouse, X: x, Y: y, Landed: mouseLanded(), Lifted: mouseLifted()})
	}

	touches.watch()

	for _, touch := range touches.down {
		pointers = append(pointers, Pointer{
			ID: PointerID(touch.id), X: touch.at.X, Y: touch.at.Y, Landed: touches.landed(touch),
		})
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

const touchJump = 64

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

type mouseLanding struct {
	at     image.Point
	key    int
	lifted bool
}

func nearTouch(at image.Point, contacts []contact) bool {
	for _, touch := range contacts {
		if distance(at, touch.at) <= mouseEchoRadius {
			return true
		}
	}
	return false
}

// deferMouse takes the mouse at its original landing, but gives a matching touch one more
// tick to arrive before pressing. A matched mouse stays taken and never holds the key.
func (k *Keyboard) deferMouse(pointer Pointer, l landing) landing {
	at := image.Pt(pointer.X, pointer.Y)
	if !nearTouch(at, touches.down) && !nearTouch(at, touches.before) {
		k.mouse = mouseLanding{at: at, key: l.key, lifted: pointer.Lifted}
		l.pending = true
	}
	l.key = noKey
	return l
}

func (k *Keyboard) resolveMouse(listening bool) {
	l := k.landed[Mouse]
	if !l.pending || !listening || k.hidden {
		return
	}

	l.pending = false
	if !nearTouch(k.mouse.at, touches.down) {
		l.key = k.mouse.key
		l = k.pressLanding(l, k.mouse.lifted)
	}
	k.landed[Mouse] = l
}
