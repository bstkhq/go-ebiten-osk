package main

import (
	"flag"

	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/bstkhq/go-ebiten-osk/hook"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var useHook = flag.Bool("hook", false, "read the on-screen keys through Ebitengine")

type typingInput struct {
	hook *hook.Hook
}

func newTypingInput(enabled bool) (typingInput, error) {
	if enabled {
		h, err := hook.New()
		return typingInput{hook: h}, err
	}
	return typingInput{}, nil
}

func (in typingInput) keyboardOptions() []osk.Option {
	if in.hook != nil {
		return []osk.Option{in.hook.Option()}
	}
	return nil
}

// readers applies the hook after the board update when the example uses it.
func (in typingInput) readers() (func(ebiten.Key) bool, func([]rune) []rune) {
	if in.hook != nil {
		in.hook.Update()
		return inpututil.IsKeyJustPressed, ebiten.AppendInputChars
	}
	return osk.IsKeyJustPressed, osk.AppendInputChars
}
