package osk_test

import (
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
type board struct {
	*osk.Keyboard
	fingers *fingers
	input   *osk.Input
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

func newBoard(t *testing.T, layout osk.Layout) board {
	t.Helper()

	b := board{fingers: &fingers{}, input: osk.NewInput()}

	keyboard, err := osk.New(layout, osk.WithPointers(b.fingers), osk.WithInput(b.input))
	if err != nil {
		t.Fatalf("a keyboard of that layout: %v", err)
	}

	keyboard.SetRect(image.Rect(0, 100, 1000, 500))
	b.Keyboard = keyboard

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

func TestTheDefaultInputHearsAKeyboardGivenNoOther(t *testing.T) {
	layout, err := osk.Stock(osk.EnUS, osk.Compact)
	if err != nil {
		t.Fatal(err)
	}

	held := &fingers{}

	keyboard, err := osk.New(layout, osk.WithPointers(held))
	if err != nil {
		t.Fatal(err)
	}

	keyboard.SetRect(image.Rect(0, 0, 1000, 400))

	b := board{Keyboard: keyboard, fingers: held}
	b.touch(t, 1, "M")
	b.Update()

	if !osk.IsKeyPressed(ebiten.KeyM) || !osk.IsKeyJustPressed(ebiten.KeyM) || osk.KeyPressDuration(ebiten.KeyM) != 1 {
		t.Error("the package's own functions did not hear the keyboard")
	}

	if got := string(osk.AppendInputChars(nil)); got != "M" {
		t.Errorf("the package's AppendInputChars gave %q, want M", got)
	}

	if got := osk.AppendPressedKeys(nil); len(got) != 1 || got[0] != ebiten.KeyM {
		t.Errorf("the package's AppendPressedKeys gave %v, want [M]", got)
	}

	if got := osk.AppendJustPressedKeys(nil); len(got) != 1 || got[0] != ebiten.KeyM {
		t.Errorf("the package's AppendJustPressedKeys gave %v, want [M]", got)
	}

	b.lift(1)
	b.Update()

	if !osk.IsKeyJustReleased(ebiten.KeyM) {
		t.Error("the package's IsKeyJustReleased did not hear the lift")
	}

	if got := osk.AppendJustReleasedKeys(nil); len(got) != 1 || got[0] != ebiten.KeyM {
		t.Errorf("the package's AppendJustReleasedKeys gave %v, want [M]", got)
	}

	b.Update()
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
