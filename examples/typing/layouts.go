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
		return g.showCompact(osk.EnUS)
	case inpututil.IsKeyJustPressed(ebiten.KeyF3):
		return g.showCompact(osk.EsES)
	case inpututil.IsKeyJustPressed(ebiten.KeyF4):
		return g.keyboard.SetLayout(pinPad())
	default:
		return nil
	}
}

func (g *game) showCompact(locale osk.Locale) error {
	layout, err := osk.Stock(locale, osk.Compact)
	if err != nil {
		return err
	}
	return g.keyboard.SetLayout(layout)
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
	ebitenutil.DebugPrintAt(screen, "F1  en-US    F3  es-ES    F4  PIN pad", 16, 36)
}
