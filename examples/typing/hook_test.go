//go:build !ebitenosk_nohook

package main

import (
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestTheExampleReadsAllThreeFieldsThroughTheHook(t *testing.T) {
	d := ebitenmcp.T(t)
	hooked, err := newGame(true)
	if err != nil {
		t.Fatal(err)
	}
	d.WithGame(func(g ebiten.Game) { *g.(*game) = *hooked })
	d.Tick(2)

	d.Click(48, boardTop+37)     // Q in Name.
	d.Click(width/2, boardTop/2) // Focus Email.
	d.Click(240, height-37)      // @ in Email.
	d.Type("a")                  // A physical character also goes to Email.
	d.Click(width-48, height-37) // Next focuses Number.
	d.Click(width/6, boardTop+37)
	d.Click(width/2, boardTop+37)
	d.Click(width/6, height-37) // Erase the second digit.

	d.WithGame(func(g ebiten.Game) {
		for index, want := range []string{"Q", "@a", "1"} {
			if got := string(g.(*game).values[index]); got != want {
				t.Errorf("%s contains %q in hook mode, want %q", fields[index].label, got, want)
			}
		}
	})
}
