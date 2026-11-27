package paint

import (
	"image"
	"unicode/utf8"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tinne26/etxt"
	"github.com/tinne26/etxt/fract"
	"golang.org/x/image/font/sfnt"
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

type labelGlyph struct {
	mask   etxt.GlyphMask
	origin fract.Point
}

// collectGlyph uses the same positioning and masks as etxt's regular draw.
func (p *Painter) collectGlyph(_ etxt.Target, index sfnt.GlyphIndex, origin fract.Point) {
	mask := p.text.Glyph().LoadMask(index, origin)
	if mask == nil {
		return
	}
	point := image.Pt(origin.X.ToIntFloor(), origin.Y.ToIntFloor())
	p.ink = p.ink.Union(mask.Bounds().Add(point))
	p.glyphs = append(p.glyphs, labelGlyph{mask, origin})
}

// drawLabel first fits the text's advance, then fits the actual ink, including overhang.
func (p *Painter) drawLabel(dst *ebiten.Image, label string, area image.Rectangle, size float64) {
	if label == "" || size == 0 {
		return
	}

	// etxt limits logical sizes to just under 16,384 pixels.
	size = min(size, 16383)
	p.text.SetSize(size)
	measured := p.text.Measure(label)
	scale := min(float64(area.Dx())/float64(measured.IntWidth()), float64(area.Dy())/float64(measured.IntHeight()))
	p.text.SetSize(size * min(1, 0.94*scale))

	middle := area.Min.Add(area.Size().Div(2))
	shift := p.fitInk(dst, label, middle, area)
	for _, glyph := range p.glyphs {
		p.text.Glyph().DrawMask(dst, glyph.mask, glyph.origin.AddPoint(shift))
	}
	p.resetGlyphs()
}

func (p *Painter) fitInk(dst *ebiten.Image, label string, middle image.Point, area image.Rectangle) fract.Point {
	text := p.text.Fract()
	for size := text.GetSize(); size > 0; size = text.GetSize() {
		p.text.Draw(dst, label, middle.X, middle.Y)
		scale := min(float64(area.Dx())/float64(p.ink.Dx()), float64(area.Dy())/float64(p.ink.Dy()))
		if scale >= 1 {
			x := max(area.Min.X-p.ink.Min.X, min(0, area.Max.X-p.ink.Max.X))
			y := max(area.Min.Y-p.ink.Min.Y, min(0, area.Max.Y-p.ink.Max.Y))
			return fract.IntsToPoint(x, y)
		}
		p.resetGlyphs()
		text.SetSize(min(size-1, fract.FromFloat64Down(size.ToFloat64()*scale)))
	}
	return fract.Point{}
}

func (p *Painter) resetGlyphs() {
	clear(p.glyphs)
	p.glyphs = p.glyphs[:0]
	p.ink = image.Rectangle{}
}
