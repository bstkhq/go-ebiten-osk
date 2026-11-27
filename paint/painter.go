package paint

import (
	"image"
	"math"

	shapes "github.com/erparts/go-shapes"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tinne26/etxt"
	"github.com/tinne26/etxt/cache"

	osk "github.com/bstkhq/go-ebiten-osk"
)

// glyphCacheBytes is room for the glyphs of a keyboard's labels at a few sizes.
const glyphCacheBytes = 4 << 20

// A Painter draws keyboards as its theme says.
type Painter struct {
	theme  Theme
	shapes *shapes.Renderer
	text   *etxt.Renderer
	keys   []osk.PlacedKey
}

// New is a painter of the theme. A theme with no font draws its labels in the stock themes'
// Go Bold.
func New(theme Theme) *Painter {
	text := etxt.NewRenderer()
	text.SetAlign(etxt.Center)

	// A label is the layout's, and a layout may be a game's own: a character the theme's
	// font lacks is drawn as that font's mark for a missing one, not a panic in Draw.
	text.Glyph().SetMissHandler(etxt.OnMissNotdef)

	painter := &Painter{shapes: shapes.NewRenderer(), text: text}
	painter.SetTheme(theme)
	return painter
}

// SetTheme replaces the theme for the next Draw. Call it from the game's Update or Draw.
func (p *Painter) SetTheme(theme Theme) {
	if theme.Font == nil {
		theme.Font = goBold()
	}

	// etxt identifies fonts by address. Keep the cache private and replace it when the
	// font changes so a newly parsed font cannot inherit a previous font's glyphs.
	if p.text.GetFont() != theme.Font {
		p.text.SetCacheHandler(cache.NewDefaultCache(glyphCacheBytes).NewHandler())
		p.text.SetFont(theme.Font)
	}

	p.theme = theme
}

// Draw draws the keyboard on dst, where its rectangle says. A hidden keyboard is not drawn.
func (p *Painter) Draw(dst *ebiten.Image, keyboard *osk.Keyboard) {
	if !keyboard.Shown() || keyboard.Rect().Empty() {
		return
	}

	p.shapes.SetColor(p.theme.Palette.Panel)
	p.shapes.DrawIntRect(dst, keyboard.Rect())

	p.keys = keyboard.AppendKeys(p.keys[:0])
	for _, key := range p.keys {
		p.drawKey(dst, key)
	}
}

func (p *Painter) drawKey(dst *ebiten.Image, key osk.PlacedKey) {
	if key.Rect.Empty() {
		return
	}

	row := float64(key.Rect.Dy())
	maxGap := float64((min(key.Rect.Dx(), key.Rect.Dy()) - 1) / 2)
	cell := key.Rect.Inset(int(math.Round(scaled(p.theme.Gap/2, row, maxGap))))
	lip := int(math.Round(scaled(p.theme.LipHeight, row, float64(cell.Dy()-1))))
	colors := p.theme.Colors(key.Role(), key.State)

	face := cell
	if key.State == osk.StatePressed {
		face.Min.Y += lip
	} else {
		face.Max.Y -= lip
		p.shapes.SetColor(colors.Lip)
		p.shapes.DrawRect(dst, cell, rounding(p.theme.Radius, row, cell))
	}

	radius := rounding(p.theme.Radius, row, face)
	p.shapes.SetColor(colors.Face)
	p.shapes.DrawRect(dst, face, radius)
	p.shapes.SetColor(colors.Border)
	border := float32(scaled(p.theme.BorderWidth, row, float64(min(face.Dx(), face.Dy()))/2))
	p.shapes.StrokeRect(dst, face, 0, border, radius)

	area := face.Inset(int(math.Ceil(scaled(p.theme.Padding, row, float64(min(face.Dx(), face.Dy()))/2))))
	p.shapes.SetColor(colors.Label)
	p.text.SetColor(colors.Label)
	p.drawLegend(dst, key, area, row)
}

func scaled(fraction, row, limit float64) float64 {
	if !(fraction > 0) {
		return 0
	}
	return min(fraction*row, limit)
}

func rounding(fraction, row float64, rect image.Rectangle) float32 {
	return float32(scaled(fraction, row, float64(min(rect.Dx(), rect.Dy()))/2))
}
