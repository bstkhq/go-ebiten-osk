package paint_test

import (
	"image/color"
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/bstkhq/go-ebiten-osk/paint"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestALayerSwitchUsesItsRoleColorsOnBothLayers(t *testing.T) {
	d := ebitenmcp.T(t)
	theme := paint.Dark()
	want := color.RGBA{0x30, 0x80, 0x70, 0xff}
	theme.Roles[osk.RoleLayerSwitch].All.Face = want
	d.WithGame(func(game ebiten.Game) { game.(*board).painter.SetTheme(theme) })
	d.Tick(1)

	at := face(t, d, "123")
	if got := colourAt(d.Screenshot(), at); got != want {
		t.Errorf("the switch to digits has face colour %v, want %v", got, want)
	}
	space := face(t, d, "Space")
	space.X -= 24 // Sample clear of the wider word label.
	if got := colourAt(d.Screenshot(), space); got != theme.Palette.Control {
		t.Errorf("the space key has face colour %v, want %v", got, theme.Palette.Control)
	}

	d.Click(float64(at.X), float64(at.Y))
	d.Tick(2)
	if got := colourAt(d.Screenshot(), face(t, d, "ABC")); got != want {
		t.Errorf("the switch back to letters has face colour %v, want %v", got, want)
	}
}
