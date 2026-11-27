package osk

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

// An Event is one thing a keyboard did in a tick, for a reader that does not poll.
type Event struct {
	Kind EventKind

	// Key is the physical position that went down or up, on a KeyDown or a KeyUp.
	Key ebiten.Key

	// Text is what was typed, on a Text.
	Text string
}

// An EventKind tells the events of a keyboard apart.
type EventKind int

const (
	// KeyDown is a physical position going down. A key that has no position of its own on
	// the locale's physical board makes no KeyDown and no KeyUp, only a Text.
	KeyDown EventKind = iota + 1
	// KeyUp is a physical position going up.
	KeyUp
	// Text is text being typed.
	Text
)

func (k EventKind) String() string {
	switch k {
	case KeyDown:
		return "key down"
	case KeyUp:
		return "key up"
	case Text:
		return "text"
	default:
		return fmt.Sprintf("EventKind(%d)", int(k))
	}
}

func (e Event) String() string {
	if e.Kind == Text {
		return fmt.Sprintf("%v %q", e.Kind, e.Text)
	}

	return fmt.Sprintf("%v %v", e.Kind, e.Key)
}
