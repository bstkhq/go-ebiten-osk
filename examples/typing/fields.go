package main

import (
	"image"
	"image/color"
	"slices"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/tinne26/etxt"
)

var fields = [...]struct {
	label  string
	kind   osk.Kind
	action osk.FieldAction
}{
	{"Name", osk.KindText, osk.ActionNext},
	{"Email", osk.KindEmail, osk.ActionNext},
	{"Number", osk.KindNumber, osk.ActionDone},
}

func fieldRect(index int) image.Rectangle {
	column := width / len(fields)
	return image.Rect(index*column+16, 103, (index+1)*column-16, 157)
}

func (g *game) chooseField() error {
	g.previousTouches, g.touches = g.touches, ebiten.AppendTouchIDs(g.previousTouches[:0])
	for _, id := range g.touches {
		if !slices.Contains(g.previousTouches, id) {
			return g.focusAt(image.Pt(ebiten.TouchPosition(id)))
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return g.focusAt(image.Pt(ebiten.CursorPosition()))
	}
	return nil
}

func (g *game) focusAt(point image.Point) error {
	for index := range fields {
		if index != g.active && point.In(fieldRect(index)) {
			return g.showStock(g.locale, g.form, index)
		}
	}
	return nil
}

func (g *game) drawFields(screen *ebiten.Image) {
	g.text.SetAlign(etxt.Left | etxt.VertCenter)
	for index, field := range fields {
		rect := fieldRect(index)
		ebitenutil.DebugPrintAt(screen, field.label, rect.Min.X, rect.Min.Y-19)
		fill := color.RGBA{0x31, 0x36, 0x40, 0xff}
		value := string(g.values[index])
		if index == g.active {
			fill = color.RGBA{0x43, 0x51, 0x66, 0xff}
			value += "_"
		}
		screen.SubImage(rect).(*ebiten.Image).Fill(fill)
		fieldImage := screen.SubImage(rect.Inset(8)).(*ebiten.Image)
		g.text.Draw(fieldImage, value, rect.Min.X+8, rect.Min.Y+rect.Dy()/2)
	}
}
