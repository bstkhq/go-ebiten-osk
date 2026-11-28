package osk_test

import (
	"slices"
	"testing"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestCustomActionKeyCallsHandlerOncePerActivation(t *testing.T) {
	layout := osk.Layout{Layers: []osk.Layer{{Rows: [][]osk.Key{
		{
			{Do: osk.Submit(osk.ActionNext), Label: "Continue"},
			{Do: osk.Type("x")},
			{Do: osk.Submit(osk.ActionSend), Label: "Send now"},
		},
	}}}}
	var actions []osk.FieldAction
	b := newBoard(t, layout, osk.WithActionHandler(func(action osk.FieldAction) {
		actions = append(actions, action)
	}))

	b.touch(t, 1, "Continue")
	b.Update()
	b.Keyboard.Update() // A second Update in the same tick must not dispatch again.
	b.touch(t, 2, "Continue")
	b.Update()
	b.lift(1)
	b.Update()
	b.lift(2)
	b.Update()
	b.tap(t, 3, "Continue")
	b.tap(t, 4, "x")
	b.tap(t, 5, "Send now")
	b.Update()

	want := []osk.FieldAction{osk.ActionNext, osk.ActionNext, osk.ActionSend}
	if !slices.Equal(actions, want) {
		t.Errorf("held press and quick taps invoked the handler with %v, want %v", actions, want)
	}
}

func TestActionHandlerCanReplaceTheLayoutAndHideTheKeyboard(t *testing.T) {
	layout, err := osk.StockWithAction(osk.EnUS, osk.Full, osk.ActionNext)
	if err != nil {
		t.Fatal(err)
	}
	replacement, err := osk.StockWithKind(osk.EnUS, osk.Full, osk.KindNumber, osk.ActionDone)
	if err != nil {
		t.Fatal(err)
	}
	var b board
	b = newBoard(t, layout, osk.WithActionHandler(func(action osk.FieldAction) {
		if !b.input.IsKeyJustPressed(ebiten.KeyEnter) {
			t.Error("Enter was not published before dispatch")
		}
		if err := b.SetLayout(replacement); err != nil {
			t.Fatal(err)
		}
		b.Hide()
	}))
	b.touch(t, 1, "Next")
	b.Update()

	want := []osk.Event{
		{Kind: osk.KeyDown, Key: ebiten.KeyEnter},
		{Kind: osk.ActionPressed, FieldAction: osk.ActionNext},
	}
	if got := b.AppendEvents(nil); !slices.Equal(got, want) {
		t.Errorf("events for Next = %v, want %v", got, want)
	}
	b.key(t, "Done")
	b.Update()
	if !b.input.IsKeyJustReleased(ebiten.KeyEnter) {
		t.Error("Enter remained held after replacing the layout and hiding the keyboard")
	}
	for range 30 {
		b.Update()
	}
	if b.Shown() {
		t.Error("the handler did not hide the keyboard")
	}
}
