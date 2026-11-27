package osk_test

import (
	"fmt"
	"image"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	osk "github.com/bstkhq/go-ebiten-osk"
)

// fingers are the pointers a test holds down, where it holds them.
type fingers []osk.Pointer

func (f *fingers) AppendPointers(pointers []osk.Pointer) []osk.Pointer {
	return append(pointers, *f...)
}

// board is a keyboard under test: its own fingers and its own input, so no test hears another.
// Its input is on a clock of its own, on which every update of the keyboard is a tick.
type board struct {
	*osk.Keyboard
	fingers *fingers
	input   *osk.Input
	next    func()
}

// Update is the next tick of the board: its clock moves on and the keyboard takes in the
// fingers. A finger has landed for the one tick it came down in, and one that lifted in it is
// gone after.
func (b board) Update() {
	b.next()
	b.Keyboard.Update()

	kept := (*b.fingers)[:0]

	for _, finger := range *b.fingers {
		if !finger.Lifted {
			finger.Landed = false
			kept = append(kept, finger)
		}
	}

	*b.fingers = kept
}

// skip is a tick in which the game does not update the board.
func (b board) skip() {
	b.next()
}

// tap is a click on the key that shows face, begun and ended between two updates.
func (b board) tap(t *testing.T, id osk.PointerID, face string) {
	t.Helper()

	b.touch(t, id, face)

	last := &(*b.fingers)[len(*b.fingers)-1]
	last.Landed, last.Lifted = true, true
}

// reland lifts finger id and puts it down again on the key that shows face, between two
// updates: the keyboard is told that it landed, as Ebitengine tells of a second quick click.
func (b board) reland(t *testing.T, id osk.PointerID, face string) {
	t.Helper()

	b.lift(id)
	b.touch(t, id, face)
	(*b.fingers)[len(*b.fingers)-1].Landed = true
}

// compactBoard is the stock compact en-US keyboard in a 1000 × 400 rectangle whose top is at 100.
func compactBoard(t *testing.T) board {
	t.Helper()

	layout, err := osk.Stock(osk.EnUS, osk.Compact)
	if err != nil {
		t.Fatalf("the stock compact en-US layout: %v", err)
	}

	return newBoard(t, layout)
}

func newBoard(t *testing.T, layout osk.Layout, opts ...osk.Option) board {
	t.Helper()

	b := board{fingers: &fingers{}}
	b.input, b.next = osk.SteppedInput()

	opts = append(opts, osk.WithPointers(b.fingers), osk.WithInput(b.input))
	keyboard, err := osk.New(layout, opts...)
	if err != nil {
		t.Fatalf("a keyboard of that layout: %v", err)
	}

	keyboard.SetRect(image.Rect(0, 100, 1000, 500))
	b.Keyboard = keyboard

	// A keyboard listens for landings from its first update on.
	b.Update()

	return b
}

// key is the key that shows face, as the keyboard holds it now.
func (b board) key(t *testing.T, face string) osk.PlacedKey {
	t.Helper()

	for _, key := range b.AppendKeys(nil) {
		if key.Face() == face {
			return key
		}
	}

	t.Fatalf("the keyboard shows no key %q", face)

	return osk.PlacedKey{}
}

// touch puts finger id on the middle of the key that shows face.
func (b board) touch(t *testing.T, id osk.PointerID, face string) {
	t.Helper()

	middle := b.key(t, face).Rect.Min.Add(b.key(t, face).Rect.Size().Div(2))
	b.touchAt(id, middle.X, middle.Y)
}

func (b board) touchAt(id osk.PointerID, x, y int) {
	*b.fingers = append(*b.fingers, osk.Pointer{ID: id, X: x, Y: y})
}

// lift takes finger id off the screen.
func (b board) lift(id osk.PointerID) {
	kept := (*b.fingers)[:0]

	for _, finger := range *b.fingers {
		if finger.ID != id {
			kept = append(kept, finger)
		}
	}

	*b.fingers = kept
}

func (b board) typed() string {
	return string(b.input.AppendInputChars(nil))
}

func TestTheCompactBoardFillsItsRectangle(t *testing.T) {
	b := compactBoard(t)

	rows := map[string]image.Rectangle{
		"Q":      image.Rect(0, 100, 100, 200),
		"P":      image.Rect(900, 100, 1000, 200),
		"A":      image.Rect(50, 200, 150, 300),
		"L":      image.Rect(850, 200, 950, 300),
		"Z":      image.Rect(150, 300, 250, 400),
		"M":      image.Rect(750, 300, 850, 400),
		"Space":  image.Rect(0, 400, 700, 500),
		"Delete": image.Rect(700, 400, 1000, 500),
	}

	for face, want := range rows {
		if got := b.key(t, face).Rect; got != want {
			t.Errorf("key %q is at %v, want %v", face, got, want)
		}
	}
}

func TestKeysNeverOverlapAtAnAwkwardWidth(t *testing.T) {
	b := compactBoard(t)
	b.SetRect(image.Rect(7, 3, 1004, 398))

	keys := b.AppendKeys(nil)

	for i, key := range keys {
		if key.Rect.Empty() || !key.Rect.In(b.Rect()) {
			t.Errorf("key %q is at %v, outside %v", key.Face(), key.Rect, b.Rect())
		}

		for _, other := range keys[i+1:] {
			if key.Rect.Overlaps(other.Rect) {
				t.Errorf("keys %q and %q overlap: %v and %v", key.Face(), other.Face(), key.Rect, other.Rect)
			}
		}
	}

	// Neighbours share their edge: a touch on the seam always belongs to a key.
	if q, w := b.key(t, "Q").Rect, b.key(t, "W").Rect; q.Max.X != w.Min.X {
		t.Errorf("Q ends at %d and W starts at %d", q.Max.X, w.Min.X)
	}
}

func TestATouchPressesItsKeyUntilItLifts(t *testing.T) {
	b := compactBoard(t)

	b.touch(t, 1, "Q")
	b.Update()

	if got := b.key(t, "Q").State; got != osk.StatePressed {
		t.Errorf("the touched key is %v, want pressed", got)
	}

	if !b.input.IsKeyPressed(ebiten.KeyQ) || !b.input.IsKeyJustPressed(ebiten.KeyQ) {
		t.Errorf("on the tick of the touch: pressed %t, just pressed %t, want both",
			b.input.IsKeyPressed(ebiten.KeyQ), b.input.IsKeyJustPressed(ebiten.KeyQ))
	}

	if got := b.typed(); got != "Q" {
		t.Errorf("the touch typed %q, want Q", got)
	}

	b.Update()
	b.Update()

	if !b.input.IsKeyPressed(ebiten.KeyQ) || b.input.IsKeyJustPressed(ebiten.KeyQ) {
		t.Error("a held key must stay pressed and stop being just pressed")
	}

	if got := b.input.KeyPressDuration(ebiten.KeyQ); got != 3 {
		t.Errorf("held for three ticks, the duration is %d", got)
	}

	if got := b.typed(); got != "" {
		t.Errorf("a held key typed %q again", got)
	}

	b.lift(1)
	b.Update()

	if b.input.IsKeyPressed(ebiten.KeyQ) || !b.input.IsKeyJustReleased(ebiten.KeyQ) {
		t.Errorf("on the tick of the lift: pressed %t, just released %t, want released",
			b.input.IsKeyPressed(ebiten.KeyQ), b.input.IsKeyJustReleased(ebiten.KeyQ))
	}

	if got := b.key(t, "Q").State; got != osk.StateIdle {
		t.Errorf("the lifted key is %v, want idle", got)
	}

	b.Update()

	if b.input.IsKeyJustReleased(ebiten.KeyQ) || b.input.KeyPressDuration(ebiten.KeyQ) != 0 {
		t.Error("a release lasts one tick")
	}
}

func TestSpaceAndEraseReportTheirPositions(t *testing.T) {
	b := compactBoard(t)

	b.touch(t, 1, "Space")
	b.touch(t, 2, "Delete")
	b.Update()

	if !b.input.IsKeyJustPressed(ebiten.KeySpace) || !b.input.IsKeyJustPressed(ebiten.KeyBackspace) {
		t.Error("space and erase must press the space bar and backspace")
	}

	if got := b.typed(); got != " " {
		t.Errorf("space and erase typed %q, want one space: erase types nothing", got)
	}
}

func TestATouchOutsideEveryKeyDoesNothing(t *testing.T) {
	b := compactBoard(t)

	// Beside the centred second row, above the board and below it.
	b.touchAt(1, 20, 250)
	b.touchAt(2, 500, 50)
	b.touchAt(3, 500, 500)
	b.Update()

	// Sliding onto a key afterwards is not landing on it.
	b.lift(1)
	b.touch(t, 1, "A")
	b.Update()

	if got := b.input.AppendPressedKeys(nil); len(got) != 0 {
		t.Errorf("touches outside every key pressed %v", got)
	}

	if got := b.typed(); got != "" {
		t.Errorf("touches outside every key typed %q", got)
	}
}

func TestAFingerOwnsTheKeyItLandedOn(t *testing.T) {
	b := compactBoard(t)

	b.touch(t, 1, "Q")
	b.Update()

	// The same finger, now over W.
	b.lift(1)
	b.touch(t, 1, "W")
	b.Update()

	if !b.input.IsKeyPressed(ebiten.KeyQ) || b.input.IsKeyPressed(ebiten.KeyW) {
		t.Error("a finger that slid from Q to W must keep Q and not press W")
	}

	if got := b.typed(); got != "" {
		t.Errorf("sliding typed %q", got)
	}
}

func TestAHiddenKeyboardTakesNothing(t *testing.T) {
	b := compactBoard(t)

	b.touch(t, 1, "Q")
	b.Update()
	b.Hide()

	if b.Shown() {
		t.Fatal("a hidden keyboard says it is shown")
	}

	b.Update()

	if !b.input.IsKeyJustReleased(ebiten.KeyQ) {
		t.Error("hiding the keyboard must release the key it held")
	}

	b.touch(t, 2, "W")
	b.Update()

	if got := b.input.AppendPressedKeys(nil); len(got) != 0 {
		t.Errorf("a hidden keyboard pressed %v", got)
	}

	b.Show()
	b.Update()

	if b.input.IsKeyPressed(ebiten.KeyQ) || b.input.IsKeyPressed(ebiten.KeyW) {
		t.Error("fingers that were down while the keyboard was hidden must not press when it shows")
	}
}

func TestWhatTheKeyboardDidIsReadAsEvents(t *testing.T) {
	b := compactBoard(t)

	b.touch(t, 1, "Q")
	b.touch(t, 2, "Delete")
	b.Update()

	want := []osk.Event{
		{Kind: osk.KeyDown, Key: ebiten.KeyQ},
		{Kind: osk.Text, Text: "Q"},
		{Kind: osk.KeyDown, Key: ebiten.KeyBackspace},
	}

	if got := b.AppendEvents(nil); !sameEvents(got, want) {
		t.Errorf("the touches were %v, want %v", got, want)
	}

	b.lift(1)
	b.Update()

	want = []osk.Event{{Kind: osk.KeyUp, Key: ebiten.KeyQ}}

	if got := b.AppendEvents(nil); !sameEvents(got, want) {
		t.Errorf("the lift was %v, want %v", got, want)
	}

	b.Update()

	if got := b.AppendEvents(nil); len(got) != 0 {
		t.Errorf("a tick in which nothing happened had the events %v", got)
	}
}

func sameEvents(got, want []osk.Event) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}

func TestKeyboardsWithTheirOwnInputsDoNotHearEachOther(t *testing.T) {
	one, other := compactBoard(t), compactBoard(t)

	one.touch(t, 1, "Q")
	one.Update()
	other.Update()

	if !one.input.IsKeyPressed(ebiten.KeyQ) {
		t.Error("a keyboard's own input did not hear it")
	}

	if other.input.IsKeyPressed(ebiten.KeyQ) || other.typed() != "" {
		t.Error("another keyboard's input heard it")
	}

	if osk.IsKeyPressed(ebiten.KeyQ) || len(osk.AppendInputChars(nil)) != 0 {
		t.Error("the default input heard a keyboard that feeds another")
	}
}

func TestWhatAFailureMessageSaysOfTheKeyboardReadsWell(t *testing.T) {
	for value, want := range map[fmt.Stringer]string{
		osk.Compact: "compact", osk.Full: "full", osk.Form(7): "Form(7)",
		osk.RoleCharacter: "character", osk.RoleSpace: "space", osk.RoleControl: "control", osk.Role(-1): "Role(-1)",
		osk.StateIdle: "idle", osk.StatePressed: "pressed", osk.State(9): "State(9)",
		osk.KeyDown: "key down", osk.KeyUp: "key up", osk.Text: "text", osk.EventKind(0): "EventKind(0)",
		osk.Event{Kind: osk.Text, Text: "ñ"}:         `text "ñ"`,
		osk.Event{Kind: osk.KeyUp, Key: ebiten.KeyQ}: "key up Q",
	} {
		if got := value.String(); got != want {
			t.Errorf("a %T prints as %q, want %q", value, got, want)
		}
	}
}

func TestAKeyOfAnotherLayoutIsReadByRoleLabelAndState(t *testing.T) {
	layout := osk.Layout{
		Positions: osk.Positions{'7': ebiten.KeyDigit7},
		Layers: []osk.Layer{{Rows: [][]osk.Key{{
			{Do: osk.Type("7")},
			{Do: osk.Type("ok"), Label: "OK", Weight: 2},
			{Do: osk.Type(" ")},
			{Do: osk.Erase(), Label: "Del"},
		}}}},
	}

	b := newBoard(t, layout)

	for face, want := range map[string]osk.Role{
		"7": osk.RoleCharacter, "OK": osk.RoleCharacter, " ": osk.RoleSpace, "Del": osk.RoleControl,
	} {
		if got := b.key(t, face).Role(); got != want {
			t.Errorf("key %q has the role %v, want %v", face, got, want)
		}
	}

	if got, want := b.key(t, "OK").Rect, image.Rect(200, 100, 600, 500); got != want {
		t.Errorf("a key of weight two is at %v, want %v", got, want)
	}

	b.touch(t, 1, "7")
	b.touch(t, 2, "OK")
	b.Update()

	if !b.input.IsKeyPressed(ebiten.KeyDigit7) {
		t.Error("a key whose character has a position in the layout must press it")
	}

	if got := b.input.AppendPressedKeys(nil); len(got) != 1 {
		t.Errorf("a key with no position pressed something: %v", got)
	}

	if got := b.typed(); got != "7ok" {
		t.Errorf("the keys typed %q, want 7ok", got)
	}
}

func TestALayoutThatCannotWorkIsRefused(t *testing.T) {
	key := osk.Key{Do: osk.Type("a")}

	broken := map[string]osk.Layout{
		"no layer":            {},
		"a layer with no row": {Layers: []osk.Layer{{}}},
		"an empty row":        {Layers: []osk.Layer{{Rows: [][]osk.Key{{key}, {}}}}},
		"does nothing":        {Layers: []osk.Layer{{Rows: [][]osk.Key{{key, {Label: "?"}}}}}},
		"a negative weight":   {Layers: []osk.Layer{{Rows: [][]osk.Key{{{Do: osk.Type("a"), Weight: -1}}}}}},
		"types nothing":       {Layers: []osk.Layer{{Rows: [][]osk.Key{{{Do: osk.Type("")}}}}}},
	}

	for name, layout := range broken {
		if _, err := osk.New(layout); err == nil {
			t.Errorf("a layout with %s was accepted", name)
		} else if !strings.Contains(err.Error(), "layer 1") && name != "no layer" {
			t.Errorf("the refusal of a layout with %s does not say where: %v", name, err)
		}
	}
}

func TestAnUnknownStockLayoutIsAnError(t *testing.T) {
	if _, err := osk.Stock("tlh-QO", osk.Compact); err == nil {
		t.Error("a locale with no stock layout was accepted")
	}

	if _, err := osk.Stock(osk.EnUS, osk.Full); err == nil {
		t.Error("a form with no stock layout yet was accepted")
	}
}

func TestEveryKeyOfTheCompactBoardTypesWhatItShows(t *testing.T) {
	shown := compactBoard(t).AppendKeys(nil)
	if len(shown) != 28 {
		t.Fatalf("the compact en-US board has %d keys, want 26 letters, space and erase", len(shown))
	}

	for _, key := range shown {
		// A board of its own for every key: nothing another key did is left in the input.
		b := compactBoard(t)
		b.touch(t, 1, key.Face())
		b.Update()

		typed, pressed := b.typed(), b.input.AppendJustPressedKeys(nil)

		want, position := key.Face(), ebiten.KeyA+ebiten.Key(key.Face()[0]-'A')

		switch key.Face() {
		case "Space":
			want, position = " ", ebiten.KeySpace
		case "Delete":
			want, position = "", ebiten.KeyBackspace
		}

		if typed != want {
			t.Errorf("the key %q typed %q, want %q", key.Face(), typed, want)
		}

		if len(pressed) != 1 || pressed[0] != position {
			t.Errorf("the key %q pressed %v, want the position %v alone", key.Face(), pressed, position)
		}
	}
}

func TestAKeyboardKeepsTheLayoutItWasMadeWith(t *testing.T) {
	layout, err := osk.Stock(osk.EnUS, osk.Compact)
	if err != nil {
		t.Fatal(err)
	}

	b := newBoard(t, layout)

	// Its maker goes on changing the layout it gave: a shorter row, another label, another
	// position.
	layout.Layers[0].Rows[0] = layout.Layers[0].Rows[0][:3]
	layout.Layers[0].Rows[1][0].Label = "changed"
	layout.Layers = append(layout.Layers[:0], osk.Layer{})
	layout.Positions['Q'] = ebiten.KeyF12

	b.SetRect(image.Rect(0, 0, 500, 200))

	if got := len(b.AppendKeys(nil)); got != 28 {
		t.Fatalf("after its maker changed the layout, the keyboard has %d keys, want its 28", got)
	}

	b.touch(t, 1, "Q")
	b.touch(t, 2, "A")
	b.Update()

	if !b.input.IsKeyPressed(ebiten.KeyQ) || b.input.IsKeyPressed(ebiten.KeyF12) || b.typed() != "QA" {
		t.Errorf("after its maker changed the layout, Q and A typed %q and pressed %v", b.typed(), b.input.AppendPressedKeys(nil))
	}
}

func TestTwoKeysOfOnePositionAreOneKeyToAGame(t *testing.T) {
	layout := osk.Layout{
		Positions: osk.Positions{'q': ebiten.KeyQ, 'Q': ebiten.KeyQ},
		Layers:    []osk.Layer{{Rows: [][]osk.Key{{{Do: osk.Type("q")}, {Do: osk.Type("Q")}}}}},
	}

	b := newBoard(t, layout)

	b.touch(t, 1, "q")
	b.Update()
	b.touch(t, 2, "Q")
	b.Update()

	want := []osk.Event{{Kind: osk.Text, Text: "Q"}}

	if got := b.AppendEvents(nil); !sameEvents(got, want) {
		t.Errorf("a second key of a position that is down made the events %v, want only its text: %v", got, want)
	}

	if b.input.IsKeyJustPressed(ebiten.KeyQ) {
		t.Error("a second key of a position that is down pressed it again")
	}

	b.lift(1)
	b.Update()

	if got := b.AppendEvents(nil); len(got) != 0 || !b.input.IsKeyPressed(ebiten.KeyQ) {
		t.Errorf("lifting one of two keys of a position made the events %v and left it pressed: %t, want none and pressed",
			got, b.input.IsKeyPressed(ebiten.KeyQ))
	}

	b.lift(2)
	b.Update()

	want = []osk.Event{{Kind: osk.KeyUp, Key: ebiten.KeyQ}}

	if got := b.AppendEvents(nil); !sameEvents(got, want) || !b.input.IsKeyJustReleased(ebiten.KeyQ) {
		t.Errorf("lifting the last key of a position made the events %v, want %v and a release", got, want)
	}
}

func TestASecondClickBetweenTwoUpdatesPressesItsOwnKey(t *testing.T) {
	b := compactBoard(t)

	b.touch(t, osk.Mouse, "H")

	for range 4 {
		b.Update()
	}

	// The mouse lets go, moves to I and presses again before the next update: one pointer
	// that never seemed to lift, unless it says it landed.
	b.reland(t, osk.Mouse, "I")
	b.Update()

	if got := b.typed(); got != "I" {
		t.Errorf("the second click typed %q, want I", got)
	}

	if !b.input.IsKeyJustReleased(ebiten.KeyH) || b.input.IsKeyPressed(ebiten.KeyH) {
		t.Error("the first click's key must go up when the second click lands elsewhere")
	}

	if !b.input.IsKeyJustPressed(ebiten.KeyI) {
		t.Error("the second click's key must go down")
	}

	want := []osk.Event{
		{Kind: osk.KeyUp, Key: ebiten.KeyH},
		{Kind: osk.KeyDown, Key: ebiten.KeyI},
		{Kind: osk.Text, Text: "I"},
	}

	if got := b.AppendEvents(nil); !sameEvents(got, want) {
		t.Errorf("the second click made the events %v, want %v", got, want)
	}

	if b.key(t, "H").State != osk.StateIdle || b.key(t, "I").State != osk.StatePressed {
		t.Errorf("after the second click H is %v and I is %v, want idle and pressed", b.key(t, "H").State, b.key(t, "I").State)
	}
}

func TestTwoQuickClicksOnOneKeyAreTwoPresses(t *testing.T) {
	b := compactBoard(t)

	b.touch(t, osk.Mouse, "Delete")
	b.Update()
	b.Update()

	if got := b.input.KeyPressDuration(ebiten.KeyBackspace); got != 2 {
		t.Fatalf("erase held for two ticks has the duration %d", got)
	}

	// Up and down again on the same key before the next update: a game that erases on a
	// press must erase again.
	b.reland(t, osk.Mouse, "Delete")
	b.Update()

	if !b.input.IsKeyJustPressed(ebiten.KeyBackspace) || !b.input.IsKeyJustReleased(ebiten.KeyBackspace) {
		t.Errorf("a second quick click on erase: just pressed %t, just released %t, want both, as Ebitengine tells of a key pressed again",
			b.input.IsKeyJustPressed(ebiten.KeyBackspace), b.input.IsKeyJustReleased(ebiten.KeyBackspace))
	}

	if !b.input.IsKeyPressed(ebiten.KeyBackspace) || b.input.KeyPressDuration(ebiten.KeyBackspace) != 1 {
		t.Errorf("a second quick click on erase: pressed %t for %d ticks, want pressed for one",
			b.input.IsKeyPressed(ebiten.KeyBackspace), b.input.KeyPressDuration(ebiten.KeyBackspace))
	}

	want := []osk.Event{{Kind: osk.KeyUp, Key: ebiten.KeyBackspace}, {Kind: osk.KeyDown, Key: ebiten.KeyBackspace}}

	if got := b.AppendEvents(nil); !sameEvents(got, want) {
		t.Errorf("a second quick click on erase made the events %v, want %v", got, want)
	}

	b.Update()

	if b.input.IsKeyJustPressed(ebiten.KeyBackspace) || b.input.IsKeyJustReleased(ebiten.KeyBackspace) || b.input.KeyPressDuration(ebiten.KeyBackspace) != 2 {
		t.Error("the tick after the second click, erase must be held with no edge, for two ticks")
	}

	// The same on a letter types it twice.
	b.lift(osk.Mouse)
	b.Update()
	b.touch(t, osk.Mouse, "A")
	b.Update()
	b.reland(t, osk.Mouse, "A")
	b.Update()

	if got := b.typed(); got != "A" {
		t.Errorf("the second quick click on a letter typed %q, want the letter again", got)
	}
}

func TestAClickBegunAndEndedBetweenTwoUpdatesIsAPressAndAReleaseInOneTick(t *testing.T) {
	b := compactBoard(t)

	b.tap(t, osk.Mouse, "H")
	b.Update()

	want := []osk.Event{
		{Kind: osk.KeyDown, Key: ebiten.KeyH},
		{Kind: osk.Text, Text: "H"},
		{Kind: osk.KeyUp, Key: ebiten.KeyH},
	}

	if got := b.AppendEvents(nil); !sameEvents(got, want) {
		t.Errorf("a whole click between two updates made the events %v, want %v", got, want)
	}

	// As Ebitengine tells of a physical key tapped within a tick: pressed, just pressed and
	// just released in it, for no time at all.
	if !b.input.IsKeyPressed(ebiten.KeyH) || !b.input.IsKeyJustPressed(ebiten.KeyH) || !b.input.IsKeyJustReleased(ebiten.KeyH) {
		t.Errorf("a whole click between two updates: pressed %t, just pressed %t, just released %t, want all three in its tick",
			b.input.IsKeyPressed(ebiten.KeyH), b.input.IsKeyJustPressed(ebiten.KeyH), b.input.IsKeyJustReleased(ebiten.KeyH))
	}

	if got := b.input.KeyPressDuration(ebiten.KeyH); got != 0 {
		t.Errorf("a whole click between two updates has the duration %d, want 0", got)
	}

	if got := b.typed(); got != "H" {
		t.Errorf("a whole click between two updates typed %q, want H", got)
	}

	if got := b.key(t, "H").State; got != osk.StateIdle {
		t.Errorf("after a whole click the key is %v, want idle", got)
	}

	b.Update()

	if b.input.IsKeyPressed(ebiten.KeyH) || b.input.IsKeyJustReleased(ebiten.KeyH) || len(b.AppendEvents(nil)) != 0 {
		t.Error("the tick after a whole click, nothing more must happen")
	}
}

// unopened is a compact board that has never been updated, with its fingers and its input.
func unopened(t *testing.T) board {
	t.Helper()

	layout, err := osk.Stock(osk.EnUS, osk.Compact)
	if err != nil {
		t.Fatal(err)
	}

	b := board{fingers: &fingers{}}
	b.input, b.next = osk.SteppedInput()

	keyboard, err := osk.New(layout, osk.WithPointers(b.fingers), osk.WithInput(b.input))
	if err != nil {
		t.Fatal(err)
	}

	keyboard.SetRect(image.Rect(0, 100, 1000, 500))
	b.Keyboard = keyboard

	return b
}

func TestAKeyboardOpenedUnderAPointerThatWasDownTypesNothing(t *testing.T) {
	b := unopened(t)

	// The press that opened the keyboard is still down, and happens to be over Q.
	b.touch(t, osk.Mouse, "Q")
	b.Update()
	b.Update()

	if got := b.typed(); got != "" || len(b.input.AppendPressedKeys(nil)) != 0 || len(b.AppendEvents(nil)) != 0 {
		t.Errorf("a keyboard opened under a pointer that was down typed %q and pressed %v", got, b.input.AppendPressedKeys(nil))
	}

	if got := b.key(t, "Q").State; got != osk.StateIdle {
		t.Errorf("the key under the pointer that was down is %v, want idle", got)
	}

	// Lifted and landed again, the same pointer types.
	b.lift(osk.Mouse)
	b.Update()
	b.touch(t, osk.Mouse, "Q")
	b.Update()

	if got := b.typed(); got != "Q" {
		t.Errorf("the pointer that landed again typed %q, want Q", got)
	}
}

func TestAKeyboardOpenedInTheTickAPointerLandsTypes(t *testing.T) {
	b := unopened(t)

	// The pointer says it landed in this very tick, as Ebitengine's mouse does.
	b.touch(t, osk.Mouse, "Q")
	(*b.fingers)[0].Landed = true
	b.Update()

	if got := b.typed(); got != "Q" || !b.input.IsKeyJustPressed(ebiten.KeyQ) {
		t.Errorf("a pointer that landed in the tick of the keyboard's first update typed %q, just pressed %t; want Q and a press",
			got, b.input.IsKeyJustPressed(ebiten.KeyQ))
	}
}

func TestAKeyboardThatMissedATickLetsGoAndTakesNoFingerThatStayedDown(t *testing.T) {
	b := compactBoard(t)

	b.touch(t, 1, "Q")
	b.Update()

	// The game leaves the keyboard out of a tick, and comes back to it with the finger where
	// it was.
	b.skip()
	b.Update()

	want := []osk.Event{{Kind: osk.KeyUp, Key: ebiten.KeyQ}}

	if got := b.AppendEvents(nil); !sameEvents(got, want) {
		t.Errorf("the keyboard that missed a tick made the events %v, want %v", got, want)
	}

	if b.input.IsKeyPressed(ebiten.KeyQ) || b.key(t, "Q").State != osk.StateIdle || b.typed() != "" {
		t.Error("a finger that stayed down through a tick the keyboard missed must hold no key")
	}
}
