package main

import (
	"image"
	"image/color"
	"math"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/tinne26/etxt"
)

var (
	customPanel  = color.RGBA{R: 0x1c, G: 0x17, B: 0x32, A: 0xff}
	customKey    = color.RGBA{R: 0x37, G: 0x2b, B: 0x58, A: 0xff}
	customAccent = color.RGBA{R: 0x73, G: 0xe8, B: 0xc3, A: 0xff}
)

// customPainter belongs to the game. Its keyboard input is the same public state any
// painter reads, while the shapes and labels here give it an appearance of its own.
type customPainter struct {
	text *etxt.Renderer
	keys []osk.PlacedKey
}

func (p *customPainter) Draw(dst *ebiten.Image, keyboard *osk.Keyboard) {
	visibility := keyboard.Visibility()
	if visibility == 0 {
		return
	}
	rect := keyboard.Rect()
	offset := image.Pt(0, int(math.Round(float64(rect.Dy())*(1-visibility))))
	board := dst.SubImage(rect).(*ebiten.Image)
	board.SubImage(rect.Add(offset)).(*ebiten.Image).Fill(customPanel)

	p.keys = keyboard.AppendKeys(p.keys[:0])
	for _, key := range p.keys {
		key.Rect = key.Rect.Add(offset)
		p.drawKey(board, key)
	}
	p.keys = keyboard.AppendVariants(p.keys[:0])
	for _, key := range p.keys {
		p.drawKey(dst, key)
	}
}

func (p *customPainter) drawKey(dst *ebiten.Image, key osk.PlacedKey) {
	tile := key.Rect.Inset(5)
	fill, label := customKey, customAccent
	if key.State != osk.StateIdle {
		fill, label = customAccent, customPanel
	}
	dst.SubImage(tile).(*ebiten.Image).Fill(fill)
	line := image.Rect(tile.Min.X, tile.Max.Y-3, tile.Max.X, tile.Max.Y)
	dst.SubImage(line).(*ebiten.Image).Fill(customAccent)

	p.text.SetColor(label)
	center := tile.Min.Add(tile.Size().Div(2))
	p.text.Draw(dst, key.Face(), center.X, center.Y)
}
