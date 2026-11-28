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
	glyphs []labelGlyph
	ink    image.Rectangle
	board  *ebiten.Image
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
	text.Glyph().SetDrawFunc(painter.collectGlyph)
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

// Draw draws the keyboard on dst. During a transition the board slides and fades
// within its rectangle.
func (p *Painter) Draw(dst *ebiten.Image, keyboard *osk.Keyboard) {
	rect, visibility := keyboard.Rect(), keyboard.Visibility()
	if visibility == 0 || rect.Empty() {
		return
	}
	if visibility == 1 {
		p.drawBoard(dst, keyboard, image.Point{})
		return
	}

	board := p.transitionImage(rect.Size())
	p.drawBoard(board, keyboard, rect.Min)

	// Fade the panel and keys together, then clip the image at the lower edge.
	options := ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	options.GeoM.Translate(float64(rect.Min.X), float64(rect.Min.Y)+float64(rect.Dy())*(1-visibility))
	options.ColorScale.ScaleAlpha(float32(visibility))
	dst.SubImage(rect).(*ebiten.Image).DrawImage(board, &options)
}

func (p *Painter) transitionImage(size image.Point) *ebiten.Image {
	if p.board != nil && p.board.Bounds().Size() != size {
		p.board.Deallocate()
		p.board = nil
	}
	if p.board == nil {
		p.board = ebiten.NewImage(size.X, size.Y)
	}
	p.board.Clear()
	return p.board
}

func (p *Painter) drawBoard(dst *ebiten.Image, keyboard *osk.Keyboard, origin image.Point) {
	p.shapes.SetColor(p.theme.Palette.Panel)
	p.shapes.DrawIntRect(dst, keyboard.Rect().Sub(origin))

	p.keys = keyboard.AppendKeys(p.keys[:0])
	for _, key := range p.keys {
		key.Rect = key.Rect.Sub(origin)
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
		p.drawLip(dst, cell, face.Max.Y, row)
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

func (p *Painter) drawLip(dst *ebiten.Image, cell image.Rectangle, top int, row float64) {
	edge := cell
	edge.Min.Y = top
	origin := dst.Bounds().Min
	edge = edge.Add(origin).Intersect(dst.Bounds())
	if edge.Empty() {
		return
	}

	// Clip the rounded base to the strip below the face. go-shapes takes coordinates
	// relative to the target's origin.
	target := dst.SubImage(edge).(*ebiten.Image)
	p.shapes.DrawRect(target, cell.Add(origin).Sub(edge.Min), rounding(p.theme.Radius, row, cell))
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
