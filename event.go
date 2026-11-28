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

	// FieldAction is what the action key says to do, on an ActionPressed event.
	FieldAction FieldAction
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
	// ActionPressed is an activation of the on-screen action key.
	ActionPressed
)

func (k EventKind) String() string {
	return named("EventKind", int(k), "", "key down", "key up", "text", "action pressed")
}

func (e Event) String() string {
	switch e.Kind {
	case Text:
		return fmt.Sprintf("%v %q", e.Kind, e.Text)
	case ActionPressed:
		return fmt.Sprintf("%v %v", e.Kind, e.FieldAction)
	default:
		return fmt.Sprintf("%v %v", e.Kind, e.Key)
	}
}
