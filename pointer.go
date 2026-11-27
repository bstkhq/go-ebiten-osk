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
	// updates, as the second of two quick clicks does. A keyboard always takes for landed a
	// pointer it did not see in the tick before, so a source that cannot tell leaves it
	// false.
	Landed bool
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

func (s *screen) AppendPointers(pointers []Pointer) []Pointer {
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		landed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
		pointers = append(pointers, Pointer{ID: Mouse, X: x, Y: y, Landed: landed})
	}

	s.touches = ebiten.AppendTouchIDs(s.touches[:0])

	for _, id := range s.touches {
		x, y := ebiten.TouchPosition(id)
		pointers = append(pointers, Pointer{ID: PointerID(id), X: x, Y: y})
	}

	return pointers
}
