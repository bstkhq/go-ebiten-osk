//go:build !ebitenosk_nohook

package main

import (
	"testing"

	ebitenmcp "github.com/bstkhq/go-ebiten-mcp"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestTheExampleReadsAllThreeFieldsThroughTheHook(t *testing.T) {
	d := hookedExample(t)

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

func TestTheExampleAdvancesAndSubmitsThroughTheHook(t *testing.T) {
	checkFormActions(t, hookedExample(t))
}

func hookedExample(t *testing.T) *ebitenmcp.Driver {
	t.Helper()
	d := ebitenmcp.T(t)
	var err error
	d.WithGame(func(g ebiten.Game) { err = g.(*game).setup(true) })
	if err != nil {
		t.Fatal(err)
	}
	d.Tick(2)
	return d
}
