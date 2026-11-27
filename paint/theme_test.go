package paint_test

import (
	"image"
	"image/color"
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/sfnt"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/bstkhq/go-ebiten-osk/paint"
)

func TestThemeColorsFollowPaletteAndOverrides(t *testing.T) {
	theme := paint.Dark()
	original := theme
	custom := color.RGBA{R: 0x85, G: 0x34, B: 0x22, A: 0xff}
	theme.Palette.Control = custom

	for _, role := range []osk.Role{osk.RoleSpace, osk.RoleControl, osk.RoleLayerSwitch, osk.Role(100)} {
		for _, state := range []osk.State{osk.StateIdle, osk.StatePressed, osk.StateLatched, osk.StateLocked} {
			before, after := original.Colors(role, state), theme.Colors(role, state)
			if after.Face == before.Face || after.Lip == before.Lip || after.Border == before.Border || after.Label != before.Label {
				t.Errorf("%v/%v did not follow the control colour: before %v, after %v", role, state, before, after)
			}
		}
	}
	for _, role := range []osk.Role{osk.RoleCharacter, osk.RoleAction} {
		if theme.Colors(role, osk.StateIdle) != original.Colors(role, osk.StateIdle) {
			t.Errorf("changing the control colour affected %v", role)
		}
	}

	theme.Roles[osk.RoleControl].All.Label = custom
	theme.States[osk.StatePressed].Face = custom
	theme.Roles[osk.RoleControl].States[osk.StatePressed].Face = color.Transparent

	pressed := theme.Colors(osk.RoleControl, osk.StatePressed)
	if pressed.Face != color.Transparent || pressed.Label != custom {
		t.Errorf("pressed control colours are %v; want a transparent face and label %v", pressed, custom)
	}
	if got := theme.Colors(osk.RoleCharacter, osk.StatePressed).Face; got != custom {
		t.Errorf("pressed character face is %v, want %v", got, custom)
	}
	if got := original.Colors(osk.RoleControl, osk.StatePressed).Label; got == custom {
		t.Errorf("changing the copy also changed the original label")
	}
}

func TestTheLightCompactBoard(t *testing.T) {
	d := ebitenmcp.T(t)
	d.WithGame(func(game ebiten.Game) { game.(*board).painter.SetTheme(paint.Light()) })
	d.Tick(1)
	d.Golden("compact_en_us_light.png")
}

func TestSwitchingThemeChangesTheNextFrame(t *testing.T) {
	d := ebitenmcp.T(t)
	g := face(t, d, "G")
	d.Tick(1)
	dark := colourAt(d.Screenshot(), g)

	light := paint.Light()
	d.WithGame(func(game ebiten.Game) { game.(*board).painter.SetTheme(light) })
	d.Tick(1)
	if got := colourAt(d.Screenshot(), g); got == dark || got != light.Palette.Key {
		t.Errorf("key after switching is %v, want %v on the next frame", got, light.Palette.Key)
	}
}

func TestChangingOnlyTheFontChangesTheLabel(t *testing.T) {
	d := ebitenmcp.T(t)
	g := face(t, d, "G")
	d.Tick(1)
	before := d.Screenshot()

	font, err := sfnt.Parse(gomono.TTF)
	if err != nil {
		t.Fatal(err)
	}
	theme := paint.Dark()
	theme.Font = font
	d.WithGame(func(game ebiten.Game) { game.(*board).painter.SetTheme(theme) })
	d.Tick(1)

	after := d.Screenshot()
	if colourAt(before, g) != colourAt(after, g) {
		t.Error("changing the font also changed the key's face colour")
	}
	if labelInk(before, theme.Palette.Label) == labelInk(after, theme.Palette.Label) {
		t.Error("switching from Go Bold to Go Mono did not change the amount of label ink")
	}
}

func TestIconsDrawEvenWithTextTurnedOff(t *testing.T) {
	d := ebitenmcp.T(t)
	theme := paint.Dark()
	theme.CharacterSize, theme.WordSize = 0, 0
	d.WithGame(func(game ebiten.Game) { game.(*board).painter.SetTheme(theme) })

	row := []osk.Key{
		{Do: osk.Erase(), Label: "Shift", Icon: osk.IconShift},
		{Do: osk.Erase(), Label: "Erase", Icon: osk.IconErase},
		{Do: osk.Erase(), Label: "Action", Icon: osk.IconAction},
		{Do: osk.Erase(), Label: "Delete"},
	}
	drawLayout(t, d, osk.Layout{Layers: []osk.Layer{{Rows: [][]osk.Key{row}}}})

	screen := d.Screenshot()
	for i, key := range row {
		cell := image.Rect(i*screenW/4, boardTop, (i+1)*screenW/4, screenH)
		inked := 0
		for y := cell.Min.Y; y < cell.Max.Y; y++ {
			for x := cell.Min.X; x < cell.Max.X; x++ {
				if screen.RGBAAt(x, y) == theme.Palette.Label {
					inked++
				}
			}
		}
		if (inked > 0) != (key.Icon != "") {
			t.Errorf("%s drew %d pixels of ink with text off", key.Label, inked)
		}
	}
}

func TestChangingOnlyTheGapShowsMorePanel(t *testing.T) {
	d := ebitenmcp.T(t)
	point := image.Pt(8, boardTop+30)
	theme := paint.Dark()
	d.Tick(1)
	if got := colourAt(d.Screenshot(), point); got != theme.Palette.Key {
		t.Fatalf("before changing the gap, got %v at the edge of Q, want key colour", got)
	}
	theme.Gap = 0.4
	d.WithGame(func(game ebiten.Game) { game.(*board).painter.SetTheme(theme) })
	d.Tick(1)
	if got := colourAt(d.Screenshot(), point); got != theme.Palette.Panel {
		t.Errorf("after changing the gap, got %v at the edge of Q, want panel colour", got)
	}
}

func TestAnOverhangingGlyphFitsInsideItsCell(t *testing.T) {
	d := ebitenmcp.T(t)
	font, err := sfnt.Parse(goitalic.TTF)
	if err != nil {
		t.Fatal(err)
	}
	theme := paint.Dark()
	theme.Font = font
	theme.CharacterSize = 0.85
	theme.Gap, theme.Padding = 0, 0

	row := []osk.Key{{Do: osk.Type(" "), Weight: 9}, {Do: osk.Type("f")}}
	drawLayout(t, d, osk.Layout{Layers: []osk.Layer{{Rows: [][]osk.Key{row}}}})
	cell := image.Rect(288, boardTop, 320, boardTop+72)
	d.WithGame(func(game ebiten.Game) {
		g := game.(*board)
		g.keyboard.SetRect(image.Rect(0, boardTop, cell.Max.X, cell.Max.Y))
		g.painter.SetTheme(theme)
	})
	d.Tick(1)

	screen := d.Screenshot()
	hasInk := false
	for y := boardTop; y < screenH; y++ {
		for x := range screenW {
			if screen.RGBAAt(x, y).R < 0x70 {
				continue
			}
			hasInk = true
			if !image.Pt(x, y).In(cell) {
				t.Fatalf("italic f has ink outside its cell at (%d, %d)", x, y)
			}
		}
	}
	if !hasInk {
		t.Error("the italic f did not draw")
	}
}

func TestATransparentFaceShowsThePanelAboveTheLip(t *testing.T) {
	d := ebitenmcp.T(t)
	theme := paint.Dark()
	theme.Roles[osk.RoleCharacter].All.Face = color.Transparent
	theme.LipHeight = 0
	setTheme := func() {
		d.WithGame(func(game ebiten.Game) { game.(*board).painter.SetTheme(theme) })
		d.Tick(1)
	}

	g := face(t, d, "G")
	edge := image.Pt(300, boardTop+115)
	setTheme()
	if got := colourAt(d.Screenshot(), g); got != theme.Palette.Panel {
		t.Errorf("transparent face with a zero-height lip shows %v, want the panel", got)
	}

	theme.LipHeight = paint.Dark().LipHeight
	setTheme()
	screen := d.Screenshot()
	if got := colourAt(screen, g); got != theme.Palette.Panel {
		t.Errorf("transparent face with a raised lip shows %v, want the panel", got)
	}
	lip := theme.Colors(osk.RoleCharacter, osk.StateIdle).Lip
	if got := colourAt(screen, edge); got != lip {
		t.Errorf("the lower edge shows %v, want the lip colour %v", got, lip)
	}
}

func drawLayout(t *testing.T, d *ebitenmcp.Driver, layout osk.Layout) {
	t.Helper()
	keyboard, err := osk.New(layout)
	if err != nil {
		t.Fatal(err)
	}
	keyboard.SetRect(image.Rect(0, boardTop, screenW, screenH))
	d.WithGame(func(game ebiten.Game) { game.(*board).keyboard = keyboard })
	d.Tick(1)
}
