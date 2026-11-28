package main

import (
	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (g *game) chooseLayout() error {
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyF1):
		return g.showStock(osk.EnUS, osk.Compact, g.active)
	case inpututil.IsKeyJustPressed(ebiten.KeyF3):
		return g.showStock(osk.EsES, osk.Compact, g.active)
	case inpututil.IsKeyJustPressed(ebiten.KeyF4):
		return g.keyboard.SetLayout(pinPad())
	case inpututil.IsKeyJustPressed(ebiten.KeyF5):
		return g.showStock(osk.EnUS, osk.Full, g.active)
	case inpututil.IsKeyJustPressed(ebiten.KeyF6):
		return g.showStock(osk.EsES, osk.Full, g.active)
	default:
		return nil
	}
}

func (g *game) showStock(locale osk.Locale, form osk.Form, index int) error {
	field := fields[index]
	boardForm := form
	if field.kind == osk.KindEmail {
		boardForm = osk.Full
	}
	layout, err := osk.StockWithKind(locale, boardForm, field.kind, field.action)
	if err != nil {
		return err
	}
	if err := g.keyboard.SetLayout(layout); err != nil {
		return err
	}
	g.locale, g.form, g.active = locale, form, index
	return nil
}

// pinPad builds a layout from the same blocks as the stock boards.
func pinPad() osk.Layout {
	positions := make(osk.Positions, 10)
	rows := make([][]osk.Key, 4)
	for i, digits := range []string{"123", "456", "789", "0"} {
		for _, digit := range digits {
			positions[digit] = ebiten.KeyDigit0 + ebiten.Key(digit-'0')
			rows[i] = append(rows[i], osk.Key{Do: osk.Type(string(digit))})
		}
	}
	rows[3] = append(rows[3], osk.Key{Do: osk.Erase(), Label: "Delete", Weight: 2})

	return osk.Layout{Positions: positions, Layers: []osk.Layer{{Rows: rows}}}
}

func drawLayoutShortcuts(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, "F1  en-US compact    F3  es-ES compact    F4  PIN pad\nF5  en-US full       F6  es-ES full", 16, 36)
}
