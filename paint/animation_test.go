package paint_test

import (
	"bytes"
	"image"
	"image/color"
	"testing"
	"time"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/bstkhq/go-ebiten-osk/paint"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestPainterSlidesAndFadesTheBoardInsideItsRectangle(t *testing.T) {
	d := ebitenmcp.T(t)
	rect := image.Rect(40, 70, 550, 290)
	theme := paint.Dark()
	theme.Palette.Panel = color.RGBA{R: 0xa0, G: 0x70, B: 0x40, A: 0xff}
	layout, err := osk.Stock(osk.EsES, osk.Compact)
	if err != nil {
		t.Fatal(err)
	}
	layerSwitch := face(t, d, "123")
	d.Click(float64(layerSwitch.X), float64(layerSwitch.Y))
	face(t, d, "1")

	pauseAfterUpdates(t, d, 2, func(b *board) {
		b.keyboard.SetAnimationDuration(0)
		b.keyboard.Hide()
		b.keyboard.SetAnimationDuration(4 * time.Second / time.Duration(ebiten.TPS()))
		b.keyboard.Show()
	})
	t.Cleanup(d.Runtime().Resume)
	hiddenAt := image.Pt(rect.Min.X+2, rect.Min.Y+8)
	visibleAt := image.Pt(rect.Min.X+2, rect.Max.Y-12)
	d.WithGame(func(game ebiten.Game) {
		b := game.(*board)
		b.keyboard.SetRect(rect)
		if err := b.keyboard.SetLayout(layout); err != nil {
			t.Error(err)
		}
		b.painter.SetTheme(theme)
	})
	face(t, d, "Ñ")

	middle := d.Screenshot()
	requireBlack(t, middle, hiddenAt)
	requireBlack(t, middle, image.Pt(rect.Min.X+2, rect.Max.Y+8))
	requireBlack(t, middle, image.Pt(rect.Min.X-2, visibleAt.Y))
	if got := colourAt(middle, visibleAt); got.R < 0x77 || got.R > 0x79 || got.G < 0x53 || got.G > 0x55 {
		t.Errorf("the board halfway through its updates has panel colour %v, want 75%% opacity", got)
	}

	d.Runtime().Resume()
	d.Tick(4)
	shown := d.Screenshot()
	if got := colourAt(shown, hiddenAt); got != theme.Palette.Panel {
		t.Errorf("the fully shown board starts with panel colour %v, want %v", got, theme.Palette.Panel)
	}

	pauseAfterUpdates(t, d, 2, func(b *board) {
		b.keyboard.Hide()
	})
	if got := d.Screenshot(); !bytes.Equal(got.Pix, middle.Pix) {
		t.Error("the board at the same point on the way out should draw the same image")
	}

	d.Runtime().Resume()
	d.Tick(4)
	requireBlack(t, d.Screenshot(), visibleAt)
}

func pauseAfterUpdates(t *testing.T, d *ebitenmcp.Driver, remaining int, begin func(*board)) {
	t.Helper()
	paused := make(chan struct{})
	d.WithGame(func(game ebiten.Game) {
		b := game.(*board)
		begin(b)
		b.afterUpdate = func() {
			remaining--
			if remaining == 0 {
				b.afterUpdate = nil
				d.Runtime().Pause()
				close(paused)
			}
		}
	})
	select {
	case <-paused:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the transition to pause")
	}
}

func requireBlack(t *testing.T, frame *image.RGBA, at image.Point) {
	t.Helper()
	if got := colourAt(frame, at); got.R != 0 || got.G != 0 || got.B != 0 {
		t.Errorf("at %v the painter drew %v, want background", at, got)
	}
}
