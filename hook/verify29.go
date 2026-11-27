//go:build !ebitenosk_nohook

package hook

import (
	"errors"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Adapted from go-ebiten-mcp v0.3.2 (MIT). See LICENSE.

func verify() error {
	if err := verifyKeys(); err != nil {
		return err
	}
	return verifyRunes()
}

func verifyKeys() error {
	// Probe near the end of both arrays, where a change to their lengths
	// is most likely to be visible. Restore the original stamps even if
	// Tab was down before the probe.
	const key = ebiten.KeyTab
	var pressed, released inputTime
	update(func(s *inputState) {
		pressed, released = s.KeyPressedTimes[key], s.KeyReleasedTimes[key]
		s.KeyPressedTimes[key], s.KeyReleasedTimes[key] = stamp(), 0
	})
	defer update(func(s *inputState) {
		s.KeyPressedTimes[key], s.KeyReleasedTimes[key] = pressed, released
	})
	if !ebiten.IsKeyPressed(key) || !inpututil.IsKeyJustPressed(key) || inpututil.KeyPressDuration(key) != 1 {
		return errors.New("press stamp did not reach Ebitengine's current tick")
	}
	update(func(s *inputState) { s.KeyReleasedTimes[key] = stamp() })
	if !inpututil.IsKeyJustReleased(key) || inpututil.KeyPressDuration(key) != 0 {
		return errors.New("release stamp did not reach Ebitengine's current tick")
	}
	return nil
}

func verifyRunes() error {
	const probe = 'ⓥ'
	var runes []rune
	update(func(s *inputState) {
		runes = s.Runes
		s.Runes = []rune{probe}
	})
	got := ebiten.AppendInputChars(nil)
	update(func(s *inputState) { s.Runes = runes })
	if len(got) != 1 || got[0] != probe {
		return errors.New("probe character did not reach ebiten.AppendInputChars")
	}
	return nil
}
