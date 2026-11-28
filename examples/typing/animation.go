package main

import (
	"time"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type keyboardVisibility struct {
	shown    bool
	animated bool
}

func (v *keyboardVisibility) update(keyboard *osk.Keyboard) {
	if inpututil.IsKeyJustPressed(ebiten.KeyF8) {
		v.animated = !v.animated
		duration := time.Duration(0)
		if v.animated {
			duration = osk.DefaultAnimationDuration
		}
		keyboard.SetAnimationDuration(duration)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF7) {
		v.shown = !v.shown
		if v.shown {
			keyboard.Show()
		} else {
			keyboard.Hide()
		}
	}
}

func (v keyboardVisibility) drawShortcuts(screen *ebiten.Image) {
	status := "off"
	if v.animated {
		status = "on"
	}
	ebitenutil.DebugPrintAt(screen, "F7  Show / hide keyboard    F8  Animation "+status, 16, 66)
}
