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
	key, err := idleProbeKey()
	if err != nil {
		return err
	}
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
	update(func(s *inputState) { s.KeyReleasedTimes[key] = stamp() + 1 })
	if !inpututil.IsKeyJustReleased(key) || inpututil.KeyPressDuration(key) != 0 {
		return errors.New("release stamp did not reach Ebitengine's current tick")
	}
	return nil
}

func idleProbeKey() (ebiten.Key, error) {
	// Start at the highest physical key so changes to the length of either
	// array are most likely to show up. Skip keys with current input.
	for key := ebiten.KeyTab; key >= 0; key-- {
		if !ebiten.IsKeyPressed(key) && !inpututil.IsKeyJustReleased(key) {
			return key, nil
		}
	}
	return 0, errors.New("no idle physical key for the input probe")
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
