package main

import (
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestTheExampleSwitchesBetweenLocalesAndItsPINPad(t *testing.T) {
	d := ebitenmcp.T(t)
	d.Tap(ebiten.KeyF1)
	d.Tick(2)
	d.Click(width-48, boardTop+112) // Ñ on the tenth cell of the second row
	d.Tap(ebiten.KeyF4)
	d.Tick(2)
	d.Click(width/6, boardTop+37) // 1 in the top left cell of the PIN pad
	d.Click(width/2, boardTop+37) // 2 beside it
	d.Tap(ebiten.KeyF3)
	d.Tick(2)
	d.Click(48, boardTop+37) // q on the full Spanish board
	d.Tick(2)

	d.WithGame(func(g ebiten.Game) {
		if got := string(g.(*game).values[0]); got != "Ñ12q" {
			t.Errorf("switching layout and typing left %q, want Ñ12q", got)
		}
	})
}

func TestTheExampleSwitchesBetweenFullAndCompact(t *testing.T) {
	d := ebitenmcp.T(t)
	d.Tap(ebiten.KeyF3)
	d.Tick(2)
	d.Click(48, boardTop+37) // q on the full English board
	d.Tap(ebiten.KeyF3)
	d.Tick(2)
	d.Click(48, boardTop+37) // Q on the compact English board
	d.Tick(2)

	d.WithGame(func(g ebiten.Game) {
		if got := string(g.(*game).values[0]); got != "qQ" {
			t.Errorf("switching forms and typing left %q, want qQ", got)
		}
	})
}

func TestTheExampleChangesLocaleAndFormIndependentlyInEmail(t *testing.T) {
	d := ebitenmcp.T(t)
	d.Click(width/2, boardTop/2)
	d.Click(48, boardTop+37) // q on the default full email board
	d.Tap(ebiten.KeyF3)
	d.Click(48, boardTop+37) // Q on the compact English board
	d.Tap(ebiten.KeyF1)
	d.Click(width-48, boardTop+112) // Ñ on the compact Spanish board
	d.Tap(ebiten.KeyF3)
	d.Click(width-48, boardTop+112) // ñ on the full Spanish board

	d.WithGame(func(g ebiten.Game) {
		if got := string(g.(*game).values[1]); got != "qQÑñ" {
			t.Errorf("the email field contains %q, want qQÑñ", got)
		}
	})
}
