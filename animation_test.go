package osk_test

import (
	"testing"
	"time"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestHideReleasesTheKeyBeforeTheBoardFinishesLeaving(t *testing.T) {
	b := compactBoard(t)
	b.touch(t, 1, "Q")
	b.Update()
	if !b.input.IsKeyJustPressed(ebiten.KeyQ) {
		t.Fatal("Q must be down before hiding")
	}

	b.Hide()
	b.Update()
	if !b.Shown() {
		t.Error("the board disappeared on the first tick of the default transition")
	}
	if b.input.IsKeyPressed(ebiten.KeyQ) || !b.input.IsKeyJustReleased(ebiten.KeyQ) {
		t.Error("the first hiding tick must release Q")
	}
}

func TestVisibilityEasesInBothDirectionsAtTheGamesTickRate(t *testing.T) {
	useTPS(t, 25)
	b := compactBoard(t)
	checkVisibility(t, b, 1, true)

	// At 25 updates per second, the default duration is five updates.
	b.Hide()
	for _, want := range []float64{0.96, 0.84, 0.64, 0.36, 0} {
		b.Update()
		checkVisibility(t, b, want, want > 0)
	}

	// A duration in between two tick boundaries rounds up to four updates.
	b.SetAnimationDuration(125 * time.Millisecond)
	b.Show()
	checkVisibility(t, b, 0, true)
	for _, want := range []float64{0.4375, 0.75, 0.9375, 1} {
		b.Update()
		checkVisibility(t, b, want, true)
		b.Keyboard.Update()
		checkVisibility(t, b, want, true)
	}
}

func TestShowAndHideReverseWithoutMovingUntilTheNextTick(t *testing.T) {
	useTPS(t, 60)
	b := animationBoard(t, 60*time.Millisecond)
	b.Hide()
	b.Update()
	checkVisibility(t, b, 0.9375, true)
	b.Hide()
	b.Update()
	checkVisibility(t, b, 0.75, true)

	b.Show()
	checkVisibility(t, b, 0.75, true)
	b.Update()
	checkVisibility(t, b, 0.9375, true)
	b.Hide()
	checkVisibility(t, b, 0.9375, true)
	b.Update()
	checkVisibility(t, b, 0.75, true)
	b.Update()
	checkVisibility(t, b, 0.4375, true)
	b.Update()
	checkVisibility(t, b, 0, false)
}

func TestAnimationCanBeDisabledAtConstructionAndDuringEitherTransition(t *testing.T) {
	useTPS(t, 60)
	b := animationBoard(t, 0)
	b.Hide()
	checkVisibility(t, b, 0, false)
	b.Show()
	checkVisibility(t, b, 1, true)

	b.SetAnimationDuration(osk.DefaultAnimationDuration)
	b.Hide()
	b.Update()
	if b.Visibility() >= 1 {
		t.Fatal("reenabling animation should ease out")
	}
	b.SetAnimationDuration(0)
	checkVisibility(t, b, 0, false)
	b.Show()
	checkVisibility(t, b, 1, true)

	b.Hide()
	b.SetAnimationDuration(osk.DefaultAnimationDuration)
	b.Show()
	b.Update()
	if b.Visibility() <= 0 || b.Visibility() >= 1 {
		t.Fatal("reenabling animation should ease in")
	}
	b.SetAnimationDuration(-time.Second)
	checkVisibility(t, b, 1, true)
}

func TestTransitionOwnsPointersWithoutTypingUntilTheyLandAgain(t *testing.T) {
	useTPS(t, 60)
	b := animationBoard(t, 60*time.Millisecond)
	b.Hide()
	b.touch(t, 1, "Delete")
	b.Update()
	requireTakenWithoutKeys(t, b, 1)

	b.Show()
	b.touch(t, 2, "123")
	b.touchAt(3, b.Rect().Max.X, b.Rect().Min.Y)
	b.Update()
	checkVisibility(t, b, 1, true)
	for _, id := range []osk.PointerID{1, 2} {
		requireTakenWithoutKeys(t, b, id)
	}
	if b.PointerTaken(3) {
		t.Error("the transition took a pointer outside its rectangle")
	}
	b.Update()
	requireTakenWithoutKeys(t, b, 1)
	requireTakenWithoutKeys(t, b, 2)

	b.reland(t, 2, "Q")
	b.Update()
	if !b.PointerTaken(2) || b.typed() != "Q" || !b.input.IsKeyJustPressed(ebiten.KeyQ) {
		t.Error("a fresh landing on the shown board should type Q")
	}
}

func TestHideThenShowInOneTickStillReleasesTheOldHold(t *testing.T) {
	b := compactBoard(t)
	b.touch(t, 1, "Q")
	b.Update()
	b.Hide()
	b.Show()
	b.Update()
	if !b.input.IsKeyJustReleased(ebiten.KeyQ) || b.input.IsKeyPressed(ebiten.KeyQ) {
		t.Error("reversing Hide before Update must still release Q")
	}
	if got := b.AppendEvents(nil); len(got) != 1 || got[0] != (osk.Event{Kind: osk.KeyUp, Key: ebiten.KeyQ}) {
		t.Errorf("events = %v, want one Q release", got)
	}
}

func useTPS(t *testing.T, rate int) {
	t.Helper()
	previous := ebiten.TPS()
	t.Cleanup(func() { ebiten.SetTPS(previous) })
	ebiten.SetTPS(rate)
}

func animationBoard(t *testing.T, duration time.Duration) board {
	t.Helper()
	layout, err := osk.Stock(osk.EnUS, osk.Compact)
	if err != nil {
		t.Fatal(err)
	}
	return newBoard(t, layout, osk.WithAnimationDuration(duration))
}

func checkVisibility(t *testing.T, b board, want float64, shown bool) {
	t.Helper()
	if got := b.Visibility(); !(got >= want-1e-12 && got <= want+1e-12) || got < 0 || got > 1 {
		t.Errorf("visibility = %v, want %v", got, want)
	} else if (want == 0 || want == 1) && got != want {
		t.Errorf("visibility = %v, want exact endpoint %v", got, want)
	}
	if b.Shown() != shown {
		t.Errorf("Shown = %t, want %t at %v visibility", b.Shown(), shown, want)
	}
}

func requireTakenWithoutKeys(t *testing.T, b board, id osk.PointerID) {
	t.Helper()
	if !b.PointerTaken(id) || b.typed() != "" || len(b.AppendEvents(nil)) != 0 {
		t.Errorf("transition pointer %d: taken %t, text %q, events %v", id,
			b.PointerTaken(id), b.typed(), b.AppendEvents(nil))
	}
}
