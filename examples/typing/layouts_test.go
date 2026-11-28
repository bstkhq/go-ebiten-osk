package main

import (
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestTheExampleSwitchesBetweenLocalesAndItsPINPad(t *testing.T) {
	d := ebitenmcp.T(t)
	d.Tap(ebiten.KeyF3)
	d.Tick(2)
	d.Click(width-48, boardTop+112) // Ñ on the tenth cell of the second row
	d.Tap(ebiten.KeyF4)
	d.Tick(2)
	d.Click(width/6, boardTop+37) // 1 in the top left cell of the PIN pad
	d.Click(width/2, boardTop+37) // 2 beside it
	d.Tap(ebiten.KeyF1)
	d.Tick(2)
	d.Click(48, boardTop+37) // Q on the letters
	d.Tick(2)

	d.WithGame(func(g ebiten.Game) {
		if got := string(g.(*game).line); got != "Ñ12Q" {
			t.Errorf("switching layout and typing left %q, want Ñ12Q", got)
		}
	})
}

func TestTheExampleSwitchesBetweenFullAndCompact(t *testing.T) {
	d := ebitenmcp.T(t)
	d.Tap(ebiten.KeyF6)
	d.Tick(2)
	d.Click(width-48, boardTop+112) // ñ on the full Spanish board
	d.Tap(ebiten.KeyF5)
	d.Tick(2)
	d.Click(48, boardTop+37) // q on the full English board
	d.Tap(ebiten.KeyF1)
	d.Tick(2)
	d.Click(48, boardTop+37) // Q on the compact English board
	d.Tick(2)

	d.WithGame(func(g ebiten.Game) {
		if got := string(g.(*game).line); got != "ñqQ" {
			t.Errorf("switching forms and typing left %q, want ñqQ", got)
		}
	})
}
