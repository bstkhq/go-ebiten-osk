package osk

// A FieldAction is what the action key asks the game to do. Every field action
// presses Enter and reports its value in an [ActionPressed] event.
type FieldAction uint8

const (
	ActionEnter FieldAction = iota
	ActionNext
	ActionDone
	ActionGo
	ActionSearch
	ActionSend
)

func (a FieldAction) String() string {
	return named("FieldAction", int(a), "enter", "next", "done", "go", "search", "send")
}

func (a FieldAction) valid() bool { return a <= ActionSend }

// Submit is the action of a key that completes the field. Give the key a Label
// for its face.
func Submit(action FieldAction) Action {
	return Action{kind: submits, field: action}
}

// WithActionHandler calls handler once per on-screen action key activation,
// in event order, at the end of [Keyboard.Update] after it feeds the input
// copy or calls the [InputWriter]. A hook still needs its own update before
// the game reads Ebitengine input.
// The handler may change the layout or visibility for subsequent updates.
// A nil handler disables the callback. ActionPressed events and Enter input
// are still emitted, and physical Enter is read through the usual input API.
func WithActionHandler(handler func(FieldAction)) Option {
	return func(o *options) { o.onAction = handler }
}

func (k *Keyboard) dispatchActions() {
	if k.onAction == nil {
		return
	}
	for _, event := range k.events {
		if event.Kind == ActionPressed {
			k.onAction(event.FieldAction)
		}
	}
}

// Shift makes the next key with lowercase text type its uppercase equivalent.
// A second press before typing locks capitals, and a third unlocks them.
// Shift changes the text and default face while keeping each key's physical position.
func Shift() Action {
	return Action{kind: shifts}
}
