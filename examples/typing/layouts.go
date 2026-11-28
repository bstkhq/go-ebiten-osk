package main

import (
	"fmt"
	"slices"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var stockLocales = osk.StockLocales()

func (g *game) chooseLayout() error {
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyF1):
		return g.cycleLocale()
	case inpututil.IsKeyJustPressed(ebiten.KeyF3):
		return g.toggleForm()
	case inpututil.IsKeyJustPressed(ebiten.KeyF4):
		return g.keyboard.SetLayout(pinPad())
	default:
		return nil
	}
}

func (g *game) cycleLocale() error {
	next := (slices.Index(stockLocales, g.locale) + 1) % len(stockLocales)
	return g.showStock(stockLocales[next], g.form, g.active)
}

func (g *game) toggleForm() error {
	form := osk.Compact
	if g.form == osk.Compact {
		form = osk.Full
	}
	if err := g.showStock(g.locale, form, g.active); err != nil {
		return err
	}
	g.preferredForm = form
	return nil
}

func (g *game) showStock(locale osk.Locale, form osk.Form, index int) error {
	field := fields[index]
	layout, err := osk.StockWithKind(locale, form, field.kind, field.action)
	if err != nil {
		return err
	}
	if err := g.keyboard.SetLayout(layout); err != nil {
		return err
	}
	g.active, g.locale, g.form = index, locale, form
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

func (g *game) drawLayoutShortcuts(screen *ebiten.Image) {
	help := fmt.Sprintf("F1  Locale (%s)    F3  Form (%s)    F4  PIN pad\nAvailable locales  %v",
		g.locale, g.form, stockLocales)
	ebitenutil.DebugPrintAt(screen, help, 16, 36)
}
