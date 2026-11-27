package paint

import (
	"image"
	"math"
	"unicode/utf8"

	shapes "github.com/erparts/go-shapes"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tinne26/etxt"
	"github.com/tinne26/etxt/cache"

	osk "github.com/bstkhq/go-ebiten-osk"
)

// glyphCacheBytes is room for the glyphs of a keyboard's labels at a few sizes.
const glyphCacheBytes = 4 << 20

// widestLabel is how much of a key's width its label may take.
const widestLabel = 0.86

// A Painter draws keyboards as its theme says.
type Painter struct {
	theme  Theme
	shapes *shapes.Renderer
	text   *etxt.Renderer
	keys   []osk.PlacedKey
}

// New is a painter of the theme.
func New(theme Theme) *Painter {
	text := etxt.NewRenderer()
	text.SetAlign(etxt.Center)

	// A label is the layout's, and a layout may be a game's own: a character the theme's
	// font lacks is drawn as that font's mark for a missing one, not a panic in Draw.
	text.Glyph().SetMissHandler(etxt.OnMissNotdef)

	// A cache of its own: etxt's shared one tells fonts apart by their address, and hands
	// a font parsed where another was freed that other's glyphs.
	text.SetCacheHandler(cache.NewDefaultCache(glyphCacheBytes).NewHandler())

	return &Painter{theme: theme, shapes: shapes.NewRenderer(), text: text}
}

// Draw draws the keyboard on dst, where its rectangle says. A hidden keyboard is not drawn.
func (p *Painter) Draw(dst *ebiten.Image, keyboard *osk.Keyboard) {
	if !keyboard.Shown() {
		return
	}

	p.shapes.SetColor(p.theme.Panel)
	p.shapes.DrawIntRect(dst, keyboard.Rect())

	p.text.SetFont(p.theme.Font)
	p.text.SetColor(p.theme.Label)

	p.keys = keyboard.AppendKeys(p.keys[:0])
	for _, key := range p.keys {
		p.drawKey(dst, key)
	}
}

// drawKey draws one key: its lip, its face over it, and its label on the face. A pressed key
// sinks onto its lip.
func (p *Painter) drawKey(dst *ebiten.Image, key osk.PlacedKey) {
	row := float64(key.Rect.Dy())
	cell := key.Rect.Inset(int(math.Round(p.theme.Gap * row / 2)))
	lip := int(math.Round(p.theme.LipHeight * row))
	radius := float32(p.theme.Radius * row)

	face := cell
	face.Max.Y -= lip
	colour := p.theme.Key

	if key.State == osk.StatePressed {
		face = cell
		face.Min.Y += lip
		colour = p.theme.Pressed
	} else {
		p.shapes.SetColor(p.theme.Lip)
		p.shapes.DrawRect(dst, cell, radius)
	}

	p.shapes.SetColor(colour)
	p.shapes.DrawRect(dst, face, radius)

	p.drawLabel(dst, key.Face(), face, row)
}

// drawLabel draws a key's label in the middle of its face, smaller if it would not fit.
func (p *Painter) drawLabel(dst *ebiten.Image, label string, face image.Rectangle, row float64) {
	size := p.theme.CharacterSize * row
	if utf8.RuneCountInString(label) > 1 {
		size = p.theme.WordSize * row
	}

	p.text.SetSize(size)

	room := widestLabel * float64(face.Dx())
	if width := float64(p.text.Measure(label).IntWidth()); width > room {
		p.text.SetSize(size * room / width)
	}

	middle := face.Min.Add(face.Size().Div(2))
	p.text.Draw(dst, label, middle.X, middle.Y)
}
