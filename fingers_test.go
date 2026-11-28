package osk_test

import (
	"slices"
	"testing"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestEraseRepeatsWithFreshEdgesWhileTwoFingersHoldIt(t *testing.T) {
	layout, err := osk.Stock(osk.EnUS, osk.Compact)
	if err != nil {
		t.Fatal(err)
	}
	b := newBoard(t, layout, osk.WithEraseRepeat(3, 2))
	b.touch(t, 1, "Delete")
	b.touch(t, 2, "Q")
	for age := 0; age <= 7; age++ {
		if age == 2 {
			b.touch(t, 3, "Delete")
		}
		if age == 4 {
			b.lift(1)
		}
		if age == 7 {
			b.lift(3)
		}
		b.Update()

		repeat := age == 3 || age == 5
		if got := b.input.IsKeyJustPressed(ebiten.KeyBackspace); got != (age == 0 || repeat) {
			t.Errorf("hold age %d: just pressed = %t", age, got)
		}
		if got := b.input.IsKeyJustReleased(ebiten.KeyBackspace); got != (age == 7 || repeat) {
			t.Errorf("hold age %d: just released = %t", age, got)
		}
		if repeat {
			want := []osk.Event{{Kind: osk.KeyUp, Key: ebiten.KeyBackspace}, {Kind: osk.KeyDown, Key: ebiten.KeyBackspace}}
			if !slices.Equal(b.AppendEvents(nil), want) || b.input.KeyPressDuration(ebiten.KeyBackspace) != 1 ||
				b.key(t, "Delete").State != osk.StatePressed {
				t.Errorf("hold age %d: repeat events %v, duration %d, state %v", age, b.AppendEvents(nil),
					b.input.KeyPressDuration(ebiten.KeyBackspace), b.key(t, "Delete").State)
			}
		}
		if age > 0 && (b.typed() != "" || b.input.IsKeyJustPressed(ebiten.KeyQ)) {
			t.Errorf("hold age %d: Q repeated", age)
		}
	}
}

func TestTakenPointersFollowTheLandingEvenInTheMargin(t *testing.T) {
	b := compactBoard(t)
	b.touchAt(1, 1, 250) // Left of the centred second row, inside the board.
	b.touchAt(2, 1, 99)
	b.touch(t, 3, "Q")
	b.Update()
	if !b.PointerTaken(1) || b.PointerTaken(2) || !b.PointerTaken(3) || b.typed() != "Q" {
		t.Errorf("initial taken: margin %t, outside %t, key %t, text %q",
			b.PointerTaken(1), b.PointerTaken(2), b.PointerTaken(3), b.typed())
	}

	(*b.fingers)[0].Y, (*b.fingers)[1].Y = 99, 150
	b.Update()
	if !b.PointerTaken(1) || b.PointerTaken(2) || b.typed() != "" {
		t.Error("sliding across the board boundary changed which fingers were taken")
	}

	b.Hide()
	b.Update()
	if b.PointerTaken(1) || b.PointerTaken(3) || !b.input.IsKeyJustReleased(ebiten.KeyQ) {
		t.Error("hiding must give up the fingers and release Q")
	}
	b.Show()
	b.Update()
	if b.PointerTaken(3) || b.input.IsKeyPressed(ebiten.KeyQ) {
		t.Error("showing under a held finger must leave it alone")
	}
}

func TestHidingKeepsThePointerTakenForTheRestOfItsTick(t *testing.T) {
	b := compactBoard(t)
	b.touch(t, 1, "Q")
	b.Update()
	if b.typed() != "Q" || !b.PointerTaken(1) {
		t.Fatalf("the keyboard typed %q and took the pointer: %t", b.typed(), b.PointerTaken(1))
	}

	b.Hide()
	if !b.PointerTaken(1) {
		t.Error("the closing pointer fell through in the tick it typed Q")
	}

	b.Update()
	if b.PointerTaken(1) || !b.input.IsKeyJustReleased(ebiten.KeyQ) {
		t.Error("the first hidden update must give up the pointer and release Q")
	}
}

func TestAnExplicitLiftFreesTheIDForTheNextLanding(t *testing.T) {
	b := compactBoard(t)
	b.tap(t, 7, "Q")
	b.Update()
	if b.typed() != "Q" || !b.PointerTaken(7) {
		t.Errorf("completed tap: typed %q, taken %t", b.typed(), b.PointerTaken(7))
	}

	b.touch(t, 7, "W")
	b.Update()
	if b.typed() != "W" || !b.input.IsKeyPressed(ebiten.KeyW) {
		t.Errorf("reused ID after explicit lift: typed %q, W down %t", b.typed(), b.input.IsKeyPressed(ebiten.KeyW))
	}
}
