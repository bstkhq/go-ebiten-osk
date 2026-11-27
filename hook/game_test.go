//go:build !ebitenosk_nohook

package hook

import (
	"context"
	"image"
	"slices"
	"testing"
	"time"

	driver "github.com/bstkhq/go-ebiten-mcp"
	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type reading struct {
	pressed, justPressed, justReleased bool
	duration                           int
	text                               string
}

type game struct {
	board   *osk.Keyboard
	hook    *Hook
	startup error
	reads   []reading
	copy    []reading
	events  []osk.Event
	pause   bool
}

func newGame() ebiten.Game { return &game{} }

func TestMain(m *testing.M) { driver.RunTests(m, newGame) }

func (g *game) Update() error {
	if g.board == nil {
		// Go test binaries omit dependency versions from runtime/debug.
		g.hook, g.startup = newHook("v2.9.9")
		layout, err := osk.Stock(osk.EnUS, osk.Compact)
		if err != nil {
			return err
		}
		var opts []osk.Option
		if g.startup == nil {
			opts = append(opts, g.hook.Option())
		}
		g.board, err = osk.New(layout, opts...)
		if err != nil {
			return err
		}
		g.board.SetRect(image.Rect(0, 0, 640, 360))
	}
	if !g.pause {
		g.board.Update()
		g.events = g.board.AppendEvents(g.events)
	}
	if g.hook != nil {
		g.hook.Update()
	}
	const h = ebiten.KeyH
	g.reads = append(g.reads, reading{
		ebiten.IsKeyPressed(h), inpututil.IsKeyJustPressed(h), inpututil.IsKeyJustReleased(h),
		inpututil.KeyPressDuration(h), string(ebiten.AppendInputChars(nil)),
	})
	g.copy = append(g.copy, reading{
		osk.IsKeyPressed(h), osk.IsKeyJustPressed(h), osk.IsKeyJustReleased(h),
		osk.KeyPressDuration(h), string(osk.AppendInputChars(nil)),
	})
	return nil
}

func (*game) Draw(*ebiten.Image)         {}
func (*game) Layout(int, int) (int, int) { return 640, 360 }

func keyCenter(t *testing.T, d *driver.Driver) image.Point {
	t.Helper()
	d.Tick(2)
	var at image.Point
	var err error
	d.WithGame(func(running ebiten.Game) {
		g := running.(*game)
		err = g.startup
		for _, key := range g.board.AppendKeys(nil) {
			if key.Face() == "H" {
				at = key.Rect.Min.Add(key.Rect.Size().Div(2))
			}
		}
	})
	if err != nil || at == (image.Point{}) {
		t.Fatalf("starting hook game: key at %v, error %v", at, err)
	}
	return at
}

func observed(t *testing.T, d *driver.Driver) (reads, copy []reading, events []osk.Event) {
	t.Helper()
	d.WithGame(func(running ebiten.Game) {
		g := running.(*game)
		reads, copy, events = slices.Clone(g.reads), slices.Clone(g.copy), slices.Clone(g.events)
	})
	return reads, copy, events
}

func TestGameReadsTouchThroughEbitengine(t *testing.T) {
	d := driver.T(t)
	at := keyCenter(t, d)
	input := d.Runtime().Injector()
	input.SetTouches([]driver.Touch{{ID: 7, X: at.X, Y: at.Y}})
	d.Tick(5)
	input.SetTouches(nil)
	d.Tick(3)
	reads, copy, events := observed(t, d)
	start := slices.Index(reads, reading{pressed: true, justPressed: true, duration: 1, text: "H"})
	if start < 0 {
		t.Fatalf("missing same-tick press and text: %v", reads)
	}
	want := []reading{
		{pressed: true, justPressed: true, duration: 1, text: "H"},
		{pressed: true, duration: 2},
		{pressed: true, duration: 3},
	}
	if len(reads) < start+len(want) || !slices.Equal(reads[start:start+len(want)], want) {
		t.Errorf("held key did not age by a tick: %v", reads)
	}
	if got := reads[len(reads)-2:]; !slices.Contains(reads[start+1:], reading{justReleased: true}) || got[1] != (reading{}) {
		t.Errorf("release is missing or persisted: %v", reads)
	}
	if !slices.Equal(copy, reads) {
		t.Errorf("input copy differs from engine: %v vs %v", copy, reads)
	}
	wantEvents := []osk.Event{{Kind: osk.KeyDown, Key: ebiten.KeyH}, {Kind: osk.Text, Text: "H"}, {Kind: osk.KeyUp, Key: ebiten.KeyH}}
	if !slices.Equal(events, wantEvents) {
		t.Errorf("keyboard events = %v, want %v", events, wantEvents)
	}
}

func TestMouseTapCanPressAndReleaseInOneTick(t *testing.T) {
	d := driver.T(t)
	at := keyCenter(t, d)
	d.Move(float64(at.X), float64(at.Y))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	input := d.Runtime().Injector()
	if err := d.Runtime().Do(ctx, func() {
		input.MouseDown(ebiten.MouseButtonLeft)
		input.MouseUp(ebiten.MouseButtonLeft)
	}); err != nil {
		t.Fatal(err)
	}
	d.Tick(3)
	reads, _, _ := observed(t, d)
	want := reading{pressed: true, justPressed: true, justReleased: true, text: "H"}
	if !slices.Contains(reads, want) || reads[len(reads)-1] != (reading{}) {
		t.Errorf("tap did not press, type and release in one tick: %v", reads)
	}
}

func TestScreenReleaseDoesNotReleasePhysicalKey(t *testing.T) {
	d := driver.T(t)
	at := keyCenter(t, d)
	input := d.Runtime().Injector()
	input.KeyDown(ebiten.KeyH)
	d.Tick(3)
	input.SetTouches([]driver.Touch{{ID: 7, X: at.X, Y: at.Y}})
	d.Tick(3)
	input.SetTouches(nil)
	d.Tick(3)
	input.KeyUp(ebiten.KeyH)
	d.Tick(3)
	reads, _, _ := observed(t, d)
	checkOneHold(t, reads)
}

func TestPhysicalHoldPassesToScreenWhenPhysicalKeyLiftsFirst(t *testing.T) {
	d := driver.T(t)
	at := keyCenter(t, d)
	input := d.Runtime().Injector()
	input.KeyDown(ebiten.KeyH)
	d.Tick(3)
	input.SetTouches([]driver.Touch{{ID: 7, X: at.X, Y: at.Y}})
	d.Tick(3)
	input.KeyUp(ebiten.KeyH)
	d.Tick(3)
	input.SetTouches(nil)
	d.Tick(3)
	reads, _, _ := observed(t, d)
	checkOneHold(t, reads)
}

func checkOneHold(t *testing.T, reads []reading) {
	t.Helper()
	var duration, releases int
	var typed string
	for _, read := range reads {
		typed += read.text
		switch {
		case read.justReleased:
			releases++
			if read.pressed || read.justPressed || read.duration != 0 {
				t.Errorf("unexpected state on release: %+v", read)
			}
		case read.pressed:
			duration++
			if read.duration != duration || read.justPressed != (duration == 1) {
				t.Errorf("hold changed at tick %d: %+v", duration, read)
			}
		}
	}
	if duration < 6 || releases != 1 || typed != "H" || reads[len(reads)-1] != (reading{}) {
		t.Errorf("expected one continuous hold, one H and one release: %v", reads)
	}
}

func TestHookReleasesAKeyboardThatStopsUpdating(t *testing.T) {
	d := driver.T(t)
	at := keyCenter(t, d)
	d.Runtime().Injector().SetTouches([]driver.Touch{{ID: 7, X: at.X, Y: at.Y}})
	d.Tick(7)
	d.WithGame(func(running ebiten.Game) { running.(*game).pause = true })
	d.Tick(3)
	reads, _, _ := observed(t, d)
	checkOneHold(t, reads)
}
