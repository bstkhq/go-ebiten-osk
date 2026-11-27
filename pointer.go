package osk

import (
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
type screen struct {
	touches []ebiten.TouchID
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

func (s *screen) AppendPointers(pointers []Pointer) []Pointer {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		pointers = append(pointers, Pointer{ID: Mouse, X: x, Y: y, Landed: mouseLanded(), Lifted: mouseLifted()})
	}

	s.touches = ebiten.AppendTouchIDs(s.touches[:0])

	for _, id := range s.touches {
		x, y := ebiten.TouchPosition(id)
		pointers = append(pointers, Pointer{ID: PointerID(id), X: x, Y: y})
	}

	return pointers
}
