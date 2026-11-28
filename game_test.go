package osk_test

import (
	"context"
	"image"
	"slices"
	"testing"
	"time"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	"github.com/hajimehoshi/ebiten/v2"

	osk "github.com/bstkhq/go-ebiten-osk"
)

// The tests of this package run beside one game loop, go-ebiten-mcp's: a test that takes its
// driver gets a fresh reader and moves it tick by tick. The other tests update keyboards of
// their own by hand and never touch the game.
func TestMain(m *testing.M) {
	ebitenmcp.RunTests(m, newReader)
}

// reader is a game with an on-screen keyboard that reads it as any game reads a keyboard: with
// the package's own functions, where it had ebiten's and inpututil's. It keeps what it read.
type reader struct {
	keyboard *osk.Keyboard

	text     []rune
	pressed  []ebiten.Key
	released []ebiten.Key

	// What it saw of H: whether it is down, the keys down with it, how long it was held at
	// most, and how many times it went up.
	hDown    bool
	down     []ebiten.Key
	hLongest int
	hUps     int

	// hTaps counts the ticks in which H went down and up at once, for no time at all.
	hTaps int
	taken []osk.PointerID

	afterUpdate func()
}

func newReader() ebiten.Game {
	layout, err := osk.Stock(osk.EnUS, osk.Compact)
	if err != nil {
		panic(err)
	}

	keyboard, err := osk.New(layout)
	if err != nil {
		panic(err)
	}

	keyboard.SetRect(image.Rect(0, 120, 640, 360))

	return &reader{keyboard: keyboard}
}

func (g *reader) Update() error {
	if g.keyboard != nil {
		g.keyboard.Update()
		g.recordTaken()
	}

	if osk.IsKeyJustPressed(ebiten.KeyBackspace) && len(g.text) > 0 {
		g.text = g.text[:len(g.text)-1]
	}

	g.text = osk.AppendInputChars(g.text)
	g.pressed = osk.AppendJustPressedKeys(g.pressed)
	g.released = osk.AppendJustReleasedKeys(g.released)

	g.hDown = osk.IsKeyPressed(ebiten.KeyH)
	g.down = osk.AppendPressedKeys(g.down[:0])
	g.hLongest = max(g.hLongest, osk.KeyPressDuration(ebiten.KeyH))

	if osk.IsKeyJustReleased(ebiten.KeyH) {
		g.hUps++
	}

	if osk.IsKeyJustPressed(ebiten.KeyH) && osk.IsKeyJustReleased(ebiten.KeyH) && osk.KeyPressDuration(ebiten.KeyH) == 0 {
		g.hTaps++
	}

	if after := g.afterUpdate; after != nil {
		g.afterUpdate = nil
		after()
	}

	return nil
}

func (g *reader) recordTaken() {
	g.taken = g.taken[:0]
	for _, touch := range ebiten.AppendTouchIDs(nil) {
		id := osk.PointerID(touch)
		if g.keyboard.PointerTaken(id) {
			g.taken = append(g.taken, id)
		}
	}
	if g.keyboard.PointerTaken(osk.Mouse) {
		g.taken = append(g.taken, osk.Mouse)
	}
}

func (g *reader) Draw(*ebiten.Image) {}

func (g *reader) Layout(int, int) (int, int) {
	return 640, 360
}

// hands are a test's hands on the running game.
type hands struct {
	*ebitenmcp.Driver
	t *testing.T
}

func sit(t *testing.T) hands {
	t.Helper()

	return hands{Driver: ebitenmcp.T(t), t: t}
}

// read gives what the game holds to look, between two ticks.
func (h hands) read(look func(*reader)) {
	h.t.Helper()

	h.WithGame(func(game ebiten.Game) { look(game.(*reader)) })
}

// settle lets the game run until what the driver did last has reached it: the driver's
// release of a button or a key is handed to the game in an update that may come after the
// tick the driver waited for.
func (h hands) settle() {
	h.t.Helper()

	h.Tick(2)
}

// middle is the middle of the key that shows face.
func (h hands) middle(face string) image.Point {
	h.t.Helper()

	var at image.Point

	found := false

	h.read(func(g *reader) {
		for _, key := range g.keyboard.AppendKeys(nil) {
			if key.Face() == face {
				at, found = key.Rect.Min.Add(key.Rect.Size().Div(2)), true
			}
		}
	})

	if !found {
		h.t.Fatalf("the keyboard shows no key %q", face)
	}

	return at
}

func (h hands) click(face string) {
	h.t.Helper()

	at := h.middle(face)
	h.Click(float64(at.X), float64(at.Y))
}

func (h hands) text() string {
	h.t.Helper()

	var text string

	h.read(func(g *reader) { text = string(g.text) })

	return text
}

func TestAGameReadsClicksOnTheKeysWithThePackagesFunctions(t *testing.T) {
	h := sit(t)

	h.click("H")
	h.click("I")
	h.settle()

	if got := h.text(); got != "HI" {
		t.Errorf("clicking H and I typed %q", got)
	}

	h.read(func(g *reader) {
		want := []ebiten.Key{ebiten.KeyH, ebiten.KeyI}

		if !slices.Equal(g.pressed, want) || !slices.Equal(g.released, want) {
			t.Errorf("the game saw %v pressed and %v released, want %v of each", g.pressed, g.released, want)
		}
	})

	h.click("Space")
	h.click("Delete")
	h.click("Delete")

	if got := h.text(); got != "H" {
		t.Errorf("a space and two erases after HI left %q, want H", got)
	}
}

func TestAGameSeesAKeyHeldOnTheScreenForAsLongAsItIsHeld(t *testing.T) {
	h := sit(t)

	at := h.middle("H")
	h.Move(float64(at.X), float64(at.Y))

	mouse := h.Runtime().Injector()
	mouse.MouseDown(ebiten.MouseButtonLeft)
	h.Tick(6)

	h.read(func(g *reader) {
		if !g.hDown || !slices.Equal(g.down, []ebiten.Key{ebiten.KeyH}) {
			t.Errorf("with the mouse held on H the game saw H down: %t, and these keys down: %v", g.hDown, g.down)
		}

		if g.hLongest < 4 || g.hUps != 0 {
			t.Errorf("after six ticks on H the game saw it held for %d ticks and released %d times", g.hLongest, g.hUps)
		}
	})

	if got := h.text(); got != "H" {
		t.Errorf("a key held for six ticks typed %q, want one H", got)
	}

	mouse.MouseUp(ebiten.MouseButtonLeft)
	h.Tick(3)

	h.read(func(g *reader) {
		if g.hDown || g.hUps != 1 || len(g.down) != 0 {
			t.Errorf("after the mouse let go the game saw H down: %t, released %d times, keys down: %v", g.hDown, g.hUps, g.down)
		}
	})
}

func TestAGameReadsASecondClickThatComesBetweenTwoUpdates(t *testing.T) {
	h := sit(t)

	first, second := h.middle("H"), h.middle("I")

	mouse := h.Runtime().Injector()
	h.Move(float64(first.X), float64(first.Y))
	mouse.MouseDown(ebiten.MouseButtonLeft)
	h.Tick(4)

	// Inside one tick, as a slow update or a low tick rate leaves room for: the button goes
	// up, the mouse moves to I, the button goes down again.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := h.Runtime().Do(ctx, func() {
		mouse.MouseUp(ebiten.MouseButtonLeft)
		mouse.MoveCursor(float64(second.X), float64(second.Y))
		mouse.MouseDown(ebiten.MouseButtonLeft)
	})
	if err != nil {
		t.Fatal(err)
	}

	h.Tick(3)

	if got := h.text(); got != "HI" {
		t.Errorf("two clicks, the second between two updates, typed %q, want HI", got)
	}

	h.read(func(g *reader) {
		if g.hDown || g.hUps != 1 || !slices.Equal(g.down, []ebiten.Key{ebiten.KeyI}) {
			t.Errorf("after the second click the game sees H down: %t, released %d times, and these keys down: %v; want H up, released once, and I down",
				g.hDown, g.hUps, g.down)
		}
	})

	mouse.MouseUp(ebiten.MouseButtonLeft)
	h.Tick(2)
}

// inOneTick does what do says between two updates of the game.
func (h hands) inOneTick(do func()) {
	h.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.Runtime().Do(ctx, do); err != nil {
		h.t.Fatal(err)
	}
}

func TestAGameSeesAClickBegunAndEndedBetweenTwoUpdatesInOneTick(t *testing.T) {
	h := sit(t)

	at := h.middle("H")
	h.Move(float64(at.X), float64(at.Y))

	mouse := h.Runtime().Injector()

	h.inOneTick(func() {
		mouse.MouseDown(ebiten.MouseButtonLeft)
		mouse.MouseUp(ebiten.MouseButtonLeft)
	})
	h.Tick(3)

	if got := h.text(); got != "H" {
		t.Errorf("a whole click between two updates typed %q, want H", got)
	}

	h.read(func(g *reader) {
		if g.hTaps != 1 || g.hUps != 1 || g.hDown {
			t.Errorf("a whole click between two updates: the game saw H go down and up in one tick %d times, up %d times, and down now: %t; want once, once, and up",
				g.hTaps, g.hUps, g.hDown)
		}
	})
}

func TestAKeyboardTheGameOpensUnderAHeldMouseTypesNothing(t *testing.T) {
	h := sit(t)

	over := h.middle("H")
	mouse := h.Runtime().Injector()

	// A press above the board, on what would be the game's own button, which opens the
	// keyboard under the cursor: the button is still down when the new keyboard first updates.
	h.read(func(g *reader) { g.keyboard = nil })
	h.Move(float64(over.X), 10)
	mouse.MouseDown(ebiten.MouseButtonLeft)
	h.Tick(2)
	h.Move(float64(over.X), float64(over.Y))

	h.read(func(g *reader) { g.keyboard = newReader().(*reader).keyboard })
	h.Tick(3)

	mouse.MouseUp(ebiten.MouseButtonLeft)
	h.Tick(2)

	h.read(func(g *reader) {
		if len(g.text) != 0 || len(g.pressed) != 0 {
			t.Errorf("a keyboard opened under a held mouse typed %q and pressed %v, want nothing", string(g.text), g.pressed)
		}
	})

	// The click after that is a click.
	h.click("H")

	if got := h.text(); got != "H" {
		t.Errorf("the first real click on the opened keyboard typed %q, want H", got)
	}
}

func TestAKeyboardTheGameOpensInTheTickOfAClickTypes(t *testing.T) {
	h := sit(t)

	over := h.middle("H")
	mouse := h.Runtime().Injector()

	h.read(func(g *reader) { g.keyboard = nil })
	h.Move(float64(over.X), float64(over.Y))

	var game *reader

	h.read(func(g *reader) { game = g })

	// The keyboard appears and the button goes down on H between the same two updates.
	h.inOneTick(func() {
		game.keyboard = newReader().(*reader).keyboard
		mouse.MouseDown(ebiten.MouseButtonLeft)
	})
	h.Tick(3)

	if got := h.text(); got != "H" {
		t.Errorf("a click in the tick the keyboard opened typed %q, want H", got)
	}

	mouse.MouseUp(ebiten.MouseButtonLeft)
	h.Tick(2)
}

func TestAFingerThatLandsInTheTickTheKeyboardOpensTypes(t *testing.T) {
	h := sit(t)

	over := h.middle("H")
	fingers := h.Runtime().Injector()

	var game *reader

	h.read(func(g *reader) { game, g.keyboard = g, nil })
	h.Tick(2)

	// The keyboard appears and a finger lands on H between the same two updates, then stays.
	h.inOneTick(func() {
		game.keyboard = newReader().(*reader).keyboard
		fingers.SetTouches([]ebitenmcp.Touch{{ID: 7, X: over.X, Y: over.Y}})
	})
	h.Tick(3)

	h.read(func(g *reader) {
		if string(g.text) != "H" || !g.hDown || g.hUps != 0 {
			t.Errorf("a finger that landed in the tick the keyboard opened: the game read %q, sees H down: %t, released %d times; want H, held, not released",
				string(g.text), g.hDown, g.hUps)
		}
	})

	fingers.SetTouches(nil)
	h.settle()

	h.read(func(g *reader) {
		if g.hDown || g.hUps != 1 {
			t.Errorf("after the finger lifted the game sees H down: %t, released %d times; want up, once", g.hDown, g.hUps)
		}
	})
}

func TestAKeyboardTheGameOpensOrComesBackToUnderAFingerTypesNothing(t *testing.T) {
	h := sit(t)

	over := h.middle("H")
	fingers := h.Runtime().Injector()

	// A finger is down where H will be before the keyboard is there.
	h.read(func(g *reader) { g.keyboard = nil })
	fingers.SetTouches([]ebitenmcp.Touch{{ID: 7, X: over.X, Y: over.Y}})
	h.Tick(3)

	var opened *osk.Keyboard

	h.read(func(g *reader) {
		g.keyboard = newReader().(*reader).keyboard
		opened = g.keyboard
	})
	h.Tick(3)

	h.read(func(g *reader) {
		if len(g.text) != 0 || len(g.pressed) != 0 {
			t.Errorf("a keyboard opened under a finger that was down typed %q and pressed %v, want nothing", string(g.text), g.pressed)
		}
	})

	// The same finger lifts and lands again: that types. Then the game leaves the keyboard
	// out of its updates for a while, with the finger down, and comes back to it.
	fingers.SetTouches(nil)
	h.settle()
	fingers.SetTouches([]ebitenmcp.Touch{{ID: 7, X: over.X, Y: over.Y}})
	h.Tick(3)

	h.read(func(g *reader) { g.keyboard = nil })
	h.Tick(3)
	h.read(func(g *reader) { g.keyboard = opened })
	h.Tick(3)

	h.read(func(g *reader) {
		if string(g.text) != "H" || g.hDown || g.hUps != 1 {
			t.Errorf("a keyboard the game came back to under a finger that stayed down: the game read %q, sees H down: %t, released %d times; want one H, up, released once",
				string(g.text), g.hDown, g.hUps)
		}
	})

	fingers.SetTouches(nil)
	h.settle()
}

func TestAKeyboardGivenNoInputAndNoPointersUsesItsOwn(t *testing.T) {
	h := sit(t)

	at := h.middle("H")

	// A game that passes what it has, and has nothing yet: the keyboard reads Ebitengine's
	// pointers and feeds the input the package's functions answer from.
	layout := stockBoard(t, osk.EnUS)
	var err error
	h.read(func(g *reader) {
		var keyboard *osk.Keyboard
		keyboard, err = osk.New(layout, osk.WithInput(nil), osk.WithPointers(nil))
		if err == nil {
			keyboard.SetRect(image.Rect(0, 120, 640, 360))
			g.keyboard = keyboard
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	h.Tick(2)

	h.Click(float64(at.X), float64(at.Y))

	if got := h.text(); got != "H" {
		t.Errorf("a click on a keyboard given no input and no pointers typed %q, want H", got)
	}
}

func TestAGameReadsTouchesOnTheKeys(t *testing.T) {
	h := sit(t)

	o, k := h.middle("O"), h.middle("K")

	fingers := h.Runtime().Injector()
	fingers.SetTouches([]ebitenmcp.Touch{{ID: 3, X: o.X, Y: o.Y}})
	h.Tick(2)
	fingers.SetTouches([]ebitenmcp.Touch{{ID: 3, X: o.X, Y: o.Y}, {ID: 4, X: k.X, Y: k.Y}})
	h.Tick(2)
	fingers.SetTouches(nil)
	h.Tick(2)

	if got := h.text(); got != "OK" {
		t.Errorf("a finger on O and then another on K typed %q", got)
	}

	h.read(func(g *reader) {
		if !slices.Contains(g.released, ebiten.KeyO) || !slices.Contains(g.released, ebiten.KeyK) {
			t.Errorf("lifting both fingers released %v, want O and K", g.released)
		}
	})
}

func TestAReusedTouchIDPressesItsNewKey(t *testing.T) {
	h := sit(t)
	hKey, gKey := h.middle("H"), h.middle("G")
	fingers := h.Runtime().Injector()

	fingers.SetTouches([]ebitenmcp.Touch{{ID: 7, X: hKey.X, Y: hKey.Y}})
	h.Tick(3)
	// The same slot reports the new contact, with no intervening empty sample.
	fingers.SetTouches([]ebitenmcp.Touch{{ID: 7, X: gKey.X, Y: gKey.Y}})
	h.Tick(3)

	h.read(func(g *reader) {
		if string(g.text) != "HG" || g.hDown || g.hUps != 1 || !slices.Equal(g.down, []ebiten.Key{ebiten.KeyG}) {
			t.Errorf("reused ID: typed %q, down %v, H releases %d", string(g.text), g.down, g.hUps)
		}
	})
	fingers.SetTouches(nil)
	h.settle()
}

func TestAMouseEchoAcrossAKeyEdgePressesOnce(t *testing.T) {
	for _, order := range []string{"together", "mouse first", "touch first"} {
		t.Run(order, func(t *testing.T) {
			h := sit(t)
			at := h.middle("H")
			// H starts at x=352. The mouse report is on G, three pixels from the touch.
			inject := h.Runtime().Injector()
			h.Move(350, float64(at.Y))
			mouse := func() { inject.MouseDown(ebiten.MouseButtonLeft) }
			touch := func() { inject.SetTouches([]ebitenmcp.Touch{{ID: 7, X: 353, Y: at.Y}}) }
			var game *reader
			h.read(func(g *reader) { game = g })
			h.inOneTick(func() {
				switch order {
				case "mouse first":
					mouse()
					game.afterUpdate = touch
				case "touch first":
					touch()
					game.afterUpdate = func() {
						inject.SetTouches(nil)
						mouse()
					}
				default:
					mouse()
					touch()
				}
			})
			h.Tick(3)
			want := ebiten.KeyH
			if order == "mouse first" {
				want = ebiten.KeyG
			}
			inject.SetTouches(nil)
			h.settle()

			h.read(func(g *reader) {
				if string(g.text) != want.String() || len(g.down) != 0 ||
					!slices.Equal(g.released, []ebiten.Key{want}) || !slices.Contains(g.taken, osk.Mouse) {
					t.Errorf("with only the mouse echo left: typed %q, down %v, releases %v, taken %v",
						string(g.text), g.down, g.released, g.taken)
				}
			})
			inject.MouseUp(ebiten.MouseButtonLeft)
			h.settle()
		})
	}
}

func TestAStockMouseTakesOverFromThePhysicalKeyWithoutNewEdges(t *testing.T) {
	h := sit(t)
	at := h.middle("H")
	inject := h.Runtime().Injector()
	h.Move(float64(at.X), float64(at.Y))
	inject.KeyDown(ebiten.KeyH)
	h.Tick(4)

	h.inOneTick(func() {
		inject.MouseDown(ebiten.MouseButtonLeft)
		inject.KeyUp(ebiten.KeyH)
	})
	h.Tick(4)
	h.read(func(g *reader) {
		if !slices.Equal(g.pressed, []ebiten.Key{ebiten.KeyH}) || g.hUps != 0 || !g.hDown {
			t.Errorf("between physical H and the mouse: presses %v, releases %v, held %t", g.pressed, g.released, g.hDown)
		}
	})
	inject.MouseUp(ebiten.MouseButtonLeft)
	h.settle()
}

func TestAMouseEchoOutsideTheBoardIsTaken(t *testing.T) {
	h := sit(t)
	x := h.middle("Q").X
	inject := h.Runtime().Injector()
	h.Move(float64(x), 119)
	h.inOneTick(func() {
		inject.SetTouches([]ebitenmcp.Touch{{ID: 7, X: x, Y: 121}})
		inject.MouseDown(ebiten.MouseButtonLeft)
	})
	h.Tick(3)
	h.read(func(g *reader) {
		if string(g.text) != "Q" || !slices.Contains(g.taken, 7) || !slices.Contains(g.taken, osk.Mouse) {
			t.Errorf("echo at the board edge: typed %q, taken %v", string(g.text), g.taken)
		}
	})
	inject.SetTouches(nil)
	inject.MouseUp(ebiten.MouseButtonLeft)
	h.settle()
}

func TestAMouseAndATouchFarApartPressTheirOwnKeys(t *testing.T) {
	h := sit(t)
	hKey, qKey := h.middle("H"), h.middle("Q")
	inject := h.Runtime().Injector()
	h.Move(float64(hKey.X), float64(hKey.Y))
	h.inOneTick(func() {
		inject.SetTouches([]ebitenmcp.Touch{{ID: 7, X: qKey.X, Y: qKey.Y}})
		inject.MouseDown(ebiten.MouseButtonLeft)
	})
	h.Tick(3)
	h.read(func(g *reader) {
		if string(g.text) != "QH" || !slices.Equal(g.down, []ebiten.Key{ebiten.KeyH, ebiten.KeyQ}) {
			t.Errorf("mouse and finger far apart: typed %q, down %v", string(g.text), g.down)
		}
	})
	inject.SetTouches(nil)
	inject.MouseUp(ebiten.MouseButtonLeft)
	h.settle()
}

func TestAMouseEchoStillMatchesAfterTwoTouchFreeTicks(t *testing.T) {
	h := sit(t)
	at := h.middle("H")
	inject := h.Runtime().Injector()
	h.Move(float64(at.X), float64(at.Y))
	var game *reader
	h.read(func(g *reader) { game = g })
	h.inOneTick(func() {
		inject.SetTouches([]ebitenmcp.Touch{{ID: 7, X: at.X, Y: at.Y}})
		game.afterUpdate = func() {
			inject.SetTouches(nil)
			game.afterUpdate = func() {
				game.afterUpdate = func() { inject.MouseDown(ebiten.MouseButtonLeft) }
			}
		}
	})
	h.Tick(5)
	h.read(func(g *reader) {
		if string(g.text) != "H" || len(g.down) != 0 || !slices.Contains(g.taken, osk.Mouse) {
			t.Errorf("echo after the touch lifted: typed %q, down %v, taken %v", string(g.text), g.down, g.taken)
		}
	})
	inject.MouseUp(ebiten.MouseButtonLeft)
	h.settle()
}

func TestAGameReadsTwoKeysAndTakesAFingerInTheMargin(t *testing.T) {
	h := sit(t)
	hKey, iKey := h.middle("H"), h.middle("I")
	fingers := h.Runtime().Injector()
	contacts := []ebitenmcp.Touch{
		{ID: 1, X: hKey.X, Y: hKey.Y}, {ID: 2, X: hKey.X, Y: hKey.Y},
		{ID: 3, X: iKey.X, Y: iKey.Y}, {ID: 4, X: 1, Y: hKey.Y},
	}
	fingers.SetTouches(contacts)
	h.Tick(3)
	h.read(func(g *reader) {
		if !slices.Equal(g.down, []ebiten.Key{ebiten.KeyH, ebiten.KeyI}) || !slices.Contains(g.taken, 4) {
			t.Errorf("with two keys and the margin touched: down %v, taken %v", g.down, g.taken)
		}
	})

	// Slide finger 2 from H to G and lift finger 1. H stays held by finger 2.
	contacts[1].X -= 34
	fingers.SetTouches(contacts[1:])
	h.Tick(3)
	h.read(func(g *reader) {
		if !g.hDown || g.hUps != 0 || slices.Contains(g.down, ebiten.KeyG) {
			t.Errorf("after one finger slides and one lifts: down %v, H releases %d", g.down, g.hUps)
		}
	})

	fingers.SetTouches(contacts[2:])
	h.Tick(3)
	h.read(func(g *reader) {
		if !slices.Equal(g.down, []ebiten.Key{ebiten.KeyI}) || g.hUps != 1 {
			t.Errorf("after the last finger on H lifts: down %v, H releases %d", g.down, g.hUps)
		}
		g.keyboard.SetAnimationDuration(0)
		g.keyboard.Hide()
	})
	h.Tick(3)
	h.read(func(g *reader) {
		if !slices.Contains(g.released, ebiten.KeyI) || len(g.taken) != 0 || len(g.down) != 0 {
			t.Errorf("after hiding: down %v, releases %v, taken %v", g.down, g.released, g.taken)
		}
		if len(g.text) != 2 || !slices.Contains(g.text, 'H') || !slices.Contains(g.text, 'I') {
			t.Errorf("typing with several fingers produced %q, want H and I once each", string(g.text))
		}
	})
	fingers.SetTouches(nil)
	h.settle()
}

func TestAClosingTouchAndItsMouseEchoDoNotReachTheGame(t *testing.T) {
	h := sit(t)
	at := h.middle("H")
	inject := h.Runtime().Injector()
	h.Move(float64(at.X), float64(at.Y))

	var game *reader
	h.read(func(g *reader) { game = g })

	var closing []osk.PointerID
	var pressed bool
	h.inOneTick(func() {
		inject.SetTouches([]ebitenmcp.Touch{{ID: 7, X: at.X, Y: at.Y}})
		inject.MouseDown(ebiten.MouseButtonLeft)
		game.afterUpdate = func() {
			pressed = osk.IsKeyJustPressed(ebiten.KeyH)
			game.keyboard.SetAnimationDuration(0)
			game.keyboard.Hide()
			game.recordTaken() // The game checks its own hit targets after closing the keyboard.
			closing = slices.Clone(game.taken)
		}
	})
	h.Tick(3)

	h.read(func(g *reader) {
		if !pressed || string(g.text) != "H" || !slices.Equal(closing, []osk.PointerID{7, osk.Mouse}) {
			t.Errorf("in the closing tick: pressed %t, typed %q, taken %v", pressed, g.text, closing)
		}
		if len(g.taken) != 0 || g.hDown || g.hUps != 1 {
			t.Errorf("after the hidden update: taken %v, H down %t, H releases %d", g.taken, g.hDown, g.hUps)
		}
	})

	inject.SetTouches(nil)
	inject.MouseUp(ebiten.MouseButtonLeft)
	h.settle()
}

func TestATransitionTakesTheTouchAndItsMouseEchoWithoutTyping(t *testing.T) {
	h := sit(t)
	at := h.middle("H")
	inject := h.Runtime().Injector()
	h.Move(float64(at.X), float64(at.Y))

	var game *reader
	h.read(func(g *reader) { game = g })
	h.inOneTick(func() {
		game.keyboard.Hide()
		inject.SetTouches([]ebitenmcp.Touch{{ID: 7, X: at.X, Y: at.Y}})
		inject.MouseDown(ebiten.MouseButtonLeft)
		game.afterUpdate = func() { game.keyboard.Show() }
	})
	h.Tick(4)

	h.read(func(g *reader) {
		if g.keyboard.Visibility() != 1 || !slices.Equal(g.taken, []osk.PointerID{7, osk.Mouse}) {
			t.Errorf("after reversing hide: visibility %v, taken pointers %v", g.keyboard.Visibility(), g.taken)
		}
		if len(g.down) != 0 || len(g.pressed) != 0 || len(g.text) != 0 {
			t.Errorf("the closing touch delivered keys %v and text %q", g.pressed, g.text)
		}
	})

	h.inOneTick(func() {
		inject.SetTouches(nil)
		inject.MouseUp(ebiten.MouseButtonLeft)
	})
	// Let the stock pointer source's three-tick mouse-echo window expire.
	h.Tick(4)
	h.Click(float64(at.X), float64(at.Y))
	if got := h.text(); got != "H" {
		t.Errorf("a fresh click typed %q, want H", got)
	}
}

func TestAMouseLayerSwitchReleasesTheOldKeyAndKeepsTheTouchTaken(t *testing.T) {
	h := sit(t)
	letter, layer := h.middle("H"), h.middle("123")
	inject := h.Runtime().Injector()
	inject.SetTouches([]ebitenmcp.Touch{{ID: 7, X: letter.X, Y: letter.Y}})
	h.Tick(3)

	h.Move(float64(layer.X), float64(layer.Y))
	h.inOneTick(func() { inject.MouseDown(ebiten.MouseButtonLeft) })
	h.Tick(3)
	h.middle("1")
	h.read(func(g *reader) {
		if string(g.text) != "H" || g.hDown || g.hUps != 1 ||
			!slices.Equal(g.taken, []osk.PointerID{7, osk.Mouse}) {
			t.Errorf("after the layer switch: typed %q, H down %t, H releases %d, taken %v",
				string(g.text), g.hDown, g.hUps, g.taken)
		}
	})

	inject.MouseUp(ebiten.MouseButtonLeft)
	inject.SetTouches(nil)
	h.settle()
}

func TestAGameErasesAgainWhileTheTouchStaysDown(t *testing.T) {
	h := sit(t)
	for range 4 {
		h.click("H")
	}
	h.settle()

	at := h.middle("Delete")
	fingers := h.Runtime().Injector()
	fingers.SetTouches([]ebitenmcp.Touch{{ID: 7, X: at.X, Y: at.Y}})
	h.Tick(10)
	if got := h.text(); got != "HHH" {
		t.Errorf("before the repeat delay the text is %q, want HHH", got)
	}
	h.Tick(28)
	if got := h.text(); got != "H" {
		t.Errorf("after two repeats the text is %q, want H", got)
	}
	fingers.SetTouches(nil)
	h.Tick(10)
	if got := h.text(); got != "H" {
		t.Errorf("after releasing erase the text is %q, want H", got)
	}
}

func TestTheSameGameStillReadsThePhysicalKeyboard(t *testing.T) {
	h := sit(t)

	h.click("H")
	h.Type("ey")
	h.Tap(ebiten.KeyBackspace)
	h.settle()

	if got := h.text(); got != "He" {
		t.Errorf("H on the screen, then e, y and backspace on the keyboard, left %q, want He", got)
	}

	h.read(func(g *reader) {
		if !slices.Contains(g.pressed, ebiten.KeyBackspace) || !slices.Contains(g.released, ebiten.KeyBackspace) {
			t.Errorf("the game saw %v pressed and %v released, want the physical backspace in both", g.pressed, g.released)
		}
	})
}

func TestAGameThatDropsItsOnlyKeyboardSeesItsKeysGoUpOnce(t *testing.T) {
	h := sit(t)

	at := h.middle("H")
	h.Move(float64(at.X), float64(at.Y))
	h.Runtime().Injector().MouseDown(ebiten.MouseButtonLeft)
	h.Tick(2)

	// The text field closes while the mouse still holds H: the game drops its keyboard and
	// goes on reading the input, with no other keyboard in its place.
	h.read(func(g *reader) { g.keyboard = nil })
	h.Tick(6)

	h.read(func(g *reader) {
		if g.hDown || len(g.down) != 0 {
			t.Errorf("six ticks after its keyboard was dropped, the game still sees H down: %t, keys down: %v", g.hDown, g.down)
		}

		if g.hUps != 1 || string(g.text) != "H" {
			t.Errorf("the dropped keyboard's H went up %d times and left the text %q, want once and one H", g.hUps, string(g.text))
		}
	})

	h.Runtime().Injector().MouseUp(ebiten.MouseButtonLeft)
	h.Tick(1)

	// The physical H afterwards is a press like any other.
	h.Tap(ebiten.KeyH)
	h.settle()

	h.read(func(g *reader) {
		want := []ebiten.Key{ebiten.KeyH, ebiten.KeyH}

		if !slices.Equal(g.pressed, want) || !slices.Equal(g.released, want) {
			t.Errorf("H on the dropped keyboard and then on the physical one: the game saw %v pressed and %v released, want %v of each",
				g.pressed, g.released, want)
		}
	})
}
