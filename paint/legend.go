package paint

import (
	"image"
	"unicode/utf8"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
)

// Strokes join points on a six-unit grid centered on the key.
type stroke struct{ x, y, toX, toY float64 }

var iconStrokes = map[osk.Icon][]stroke{
	osk.IconShift: {
		{-3, 0, 0, -3}, {0, -3, 3, 0}, {0, -3, 0, 3},
	},
	osk.IconErase: {
		{-3, 0, -1, -2}, {-1, -2, 3, -2}, {3, -2, 3, 2}, {3, 2, -1, 2}, {-1, 2, -3, 0},
		{0, -1, 2, 1}, {2, -1, 0, 1},
	},
	osk.IconAction: {
		{3, -2, 3, 1}, {3, 1, -3, 1}, {-3, 1, -1, -1}, {-3, 1, -1, 3},
	},
}

func (p *Painter) drawLegend(dst *ebiten.Image, key osk.PlacedKey, area image.Rectangle, row float64) {
	if area.Empty() {
		return
	}

	if strokes := iconStrokes[key.Icon]; len(strokes) > 0 {
		p.drawIcon(dst, area, strokes)
		return
	}

	label := key.Face()
	fraction := p.theme.CharacterSize
	if utf8.RuneCountInString(label) > 1 {
		fraction = p.theme.WordSize
	}
	p.drawLabel(dst, label, area, scaled(fraction, row, float64(area.Dy())))
}

func (p *Painter) drawIcon(dst *ebiten.Image, area image.Rectangle, strokes []stroke) {
	scale := float64(min(area.Dx(), area.Dy())) / 14
	x := float64(area.Min.X) + float64(area.Dx())/2
	y := float64(area.Min.Y) + float64(area.Dy())/2
	for _, line := range strokes {
		p.shapes.DrawLine(dst, x+line.x*scale, y+line.y*scale, x+line.toX*scale, y+line.toY*scale, scale*0.65)
	}
}

// drawLabel fits the measured label in the padded area. Leave room for glyph overhang.
func (p *Painter) drawLabel(dst *ebiten.Image, label string, area image.Rectangle, size float64) {
	if label == "" || size == 0 {
		return
	}

	p.text.SetSize(size)
	measured := p.text.Measure(label)
	width := float64(area.Dx()) * 0.94
	height := float64(area.Dy()) * 0.94
	if scale := min(width/float64(measured.IntWidth()), height/float64(measured.IntHeight())); scale < 1 {
		p.text.SetSize(size * scale)
	}

	middle := area.Min.Add(area.Size().Div(2))
	p.text.Draw(dst, label, middle.X, middle.Y)
}
