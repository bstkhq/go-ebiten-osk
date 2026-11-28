// This game runs as an ordinary executable so hook.New sees the linked
// dependency versions in its Go build information.
package main

import (
	"fmt"
	"image"
	"log"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/bstkhq/go-ebiten-osk/hook"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type game struct {
	keyboard *osk.Keyboard
	hook     *hook.Hook
	tick     int
}

func (g *game) AppendPointers(pointers []osk.Pointer) []osk.Pointer {
	if g.tick <= 1 {
		pointers = append(pointers, osk.Pointer{ID: 1, X: 20, Y: 20, Landed: g.tick == 0})
	}
	return pointers
}

func (g *game) Update() error {
	g.keyboard.Update()
	g.hook.Update()

	const q = ebiten.KeyQ
	pressed, down, up := ebiten.IsKeyPressed(q), inpututil.IsKeyJustPressed(q), inpututil.IsKeyJustReleased(q)
	duration, chars := inpututil.KeyPressDuration(q), string(ebiten.AppendInputChars(nil))
	wantDuration := [3]int{1, 2, 0}
	wantChars := ""
	if g.tick == 0 {
		wantChars = "Q"
	}
	if pressed != (g.tick < 2) || down != (g.tick == 0) || up != (g.tick == 2) ||
		duration != wantDuration[g.tick] || chars != wantChars {
		return fmt.Errorf("tick %d reported pressed=%t, down=%t, up=%t, duration=%d, text=%q", g.tick, pressed, down, up, duration, chars)
	}
	g.tick++
	if g.tick == 3 {
		return ebiten.Termination
	}
	return nil
}

func (*game) Draw(*ebiten.Image)         {}
func (*game) Layout(int, int) (int, int) { return 640, 360 }

func run() error {
	h, err := hook.New()
	if err != nil {
		return err
	}
	layout, err := osk.Stock(osk.EnUS, osk.Compact)
	if err != nil {
		return err
	}
	g := &game{hook: h}
	g.keyboard, err = osk.New(layout, h.Option(), osk.WithPointers(g))
	if err != nil {
		return err
	}
	g.keyboard.SetRect(image.Rect(0, 0, 640, 360))
	return ebiten.RunGame(g)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
