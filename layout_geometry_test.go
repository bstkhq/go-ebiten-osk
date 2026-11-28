package osk_test

import (
	"image"
	"math"
	"testing"

	osk "github.com/bstkhq/go-ebiten-osk"
)

func TestCustomRowGapsReserveSpaceWithoutTakingKeyWidth(t *testing.T) {
	a, b := osk.Key{Do: osk.Type("a"), Weight: 2.5}, osk.Key{Do: osk.Type("b")}
	gaps := []float64{0.5, 2, 1}
	layout := osk.Layout{Layers: []osk.Layer{{Rows: [][]osk.Key{{a, b}}, RowGaps: [][]float64{gaps}}}}
	keyboard := newBoard(t, layout)
	gaps[1] = 0
	keyboard.SetRect(image.Rect(0, 0, 700, 100))

	if got := keyboard.key(t, "a").Rect; got != image.Rect(50, 0, 300, 100) {
		t.Errorf("a is at %v, want (50,0)-(300,100)", got)
	}
	if got := keyboard.key(t, "b").Rect; got != image.Rect(500, 0, 600, 100) {
		t.Errorf("b is at %v, want (500,0)-(600,100)", got)
	}
	if got := len(keyboard.AppendKeys(nil)); got != 2 {
		t.Errorf("got %d placed keys, want 2", got)
	}
	keyboard.touchAt(1, 400, 50)
	keyboard.Update()
	if got := keyboard.AppendEvents(nil); len(got) > 0 || !keyboard.PointerTaken(1) {
		t.Errorf("touch on gap reported events %v and taken %t", got, keyboard.PointerTaken(1))
	}
}

func TestInvalidRowGapsAreRejected(t *testing.T) {
	keys := [][]osk.Key{{{Do: osk.Type("a")}}}
	for _, gaps := range [][][]float64{
		{{0, 0}, {0, 0}}, {{0}}, {{-1, 0}}, {{0, math.NaN()}}, {{0, math.Inf(1)}},
	} {
		if _, err := osk.New(osk.Layout{Layers: []osk.Layer{{Rows: keys, RowGaps: gaps}}}); err == nil {
			t.Errorf("accepted invalid row gaps %v", gaps)
		}
	}
}
