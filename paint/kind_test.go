package paint_test

import (
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	osk "github.com/bstkhq/go-ebiten-osk"
)

func TestTheNumberPadInTheStockDarkTheme(t *testing.T) {
	d := ebitenmcp.T(t)
	layout, err := osk.StockWithKind(osk.EnUS, osk.Compact, osk.KindNumber, osk.ActionDone)
	if err != nil {
		t.Fatal(err)
	}
	drawLayout(t, d, layout)
	d.Golden("number_en_us_dark.png")
}
