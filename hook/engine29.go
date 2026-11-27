//go:build !ebitenosk_nohook

package hook

import (
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2"
)

// Adapted from go-ebiten-mcp v0.3.2 (MIT). See LICENSE.

// Supported reports whether the engine integration was included in this build.
// New also checks the linked version and whether the integration works.
const Supported = true

var backend = engineBackend{verify: verify, apply: apply}

//go:linkname inputstateGet github.com/hajimehoshi/ebiten/v2/internal/inputstate.Get
func inputstateGet() unsafe.Pointer

//go:linkname inputstateUpdate github.com/hajimehoshi/ebiten/v2/internal/inputstate.(*inputState).Update
func inputstateUpdate(state unsafe.Pointer, write func(unsafe.Pointer))

// inputState mirrors the prefix of ui.InputState through Runes. Only the key
// arrays and Runes are written by the hook.
type inputState struct {
	KeyPressedTimes          [ebiten.KeyMax + 1]inputTime
	KeyReleasedTimes         [ebiten.KeyMax + 1]inputTime
	MouseButtonPressedTimes  [ebiten.MouseButtonMax + 1]inputTime
	MouseButtonReleasedTimes [ebiten.MouseButtonMax + 1]inputTime
	CursorX, CursorY         float64
	WheelX, WheelY           float64
	Touches                  []touch
	Runes                    []rune
}

type touch struct {
	ID, X, Y int
}

// Public input reads take the same lock, so they must run outside write.
func update(write func(*inputState)) {
	inputstateUpdate(inputstateGet(), func(p unsafe.Pointer) {
		write((*inputState)(p))
	})
}

func apply(h *Hook) {
	update(func(s *inputState) {
		for key, next := range h.pending.keys {
			h.heldFrom[key] = next.apply(h.heldFrom[key], &s.KeyPressedTimes[key], &s.KeyReleasedTimes[key])
		}
		s.Runes = append(s.Runes, h.pending.runes...)
	})
}
