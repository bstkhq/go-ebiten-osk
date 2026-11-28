package paint_test

import (
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/bstkhq/go-ebiten-osk/paint"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestTheFullBoardInBothStockThemes(t *testing.T) {
	for _, test := range []struct {
		name  string
		theme paint.Theme
	}{
		{"dark", paint.Dark()}, {"light", paint.Light()},
	} {
		t.Run(test.name, func(t *testing.T) {
			d := ebitenmcp.T(t)
			layout, err := osk.Stock(osk.EsES, osk.Full)
			if err != nil {
				t.Fatal(err)
			}
			drawLayout(t, d, layout)
			d.WithGame(func(game ebiten.Game) { game.(*board).painter.SetTheme(test.theme) })
			d.Tick(1)
			d.Golden("full_es_es_" + test.name + ".png")
		})
	}
}

func TestFullShiftLooksLatchedAndLocked(t *testing.T) {
	d := ebitenmcp.T(t)
	layout, err := osk.Stock(osk.EnUS, osk.Full)
	if err != nil {
		t.Fatal(err)
	}
	drawLayout(t, d, layout)

	at := face(t, d, "Shift")
	for _, state := range []osk.State{osk.StateLatched, osk.StateLocked, osk.StateIdle} {
		d.Click(float64(at.X), float64(at.Y))
		d.Tick(2)
		want := paint.Dark().Colors(osk.RoleControl, state).Face
		if got := colourAt(d.Screenshot(), at); got != want {
			t.Errorf("shift in state %v has colour %v, want %v", state, got, want)
		}
	}
}
