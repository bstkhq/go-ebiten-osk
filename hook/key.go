//go:build !ebitenosk_nohook

package hook

import "github.com/hajimehoshi/ebiten/v2"

func (k keyUpdate) apply(previous inputTime, pressed, released *inputTime) inputTime {
	if *pressed > *released {
		// The backend (or a test driver) is still holding this key. Its
		// stamps take priority. Remember its press for a handoff to the
		// on-screen key if the physical key lifts first.
		if k.holders > 0 {
			return *pressed
		}
		return 0
	}
	if k.holders > 0 {
		return k.hold(previous, pressed, released)
	}
	if k.pressed {
		*pressed = stamp()
		*released = stamp()
	} else if previous != 0 {
		*pressed = previous
		*released = stamp()
	}
	return 0
}

func (k keyUpdate) hold(previous inputTime, pressed, released *inputTime) inputTime {
	switch {
	case previous == 0:
		*pressed, *released = startHold(*pressed, *released)
	case k.restarted == k.holders:
		*released = stamp()
		*pressed = stamp()
	default:
		// The backend restored the hardware snapshot at the start
		// of this tick. Restore the ongoing press and hide any physical
		// release during that hold.
		*pressed = previous
		*released = 0
	}
	return *pressed
}

func startHold(pressed, released inputTime) (inputTime, inputTime) {
	tick := ebiten.Tick()
	if pressed != 0 && pressed.tick() < tick && released.tick() == tick {
		// A physical hold handed off to a new on-screen hold in this tick.
		return pressed, 0
	}
	return stamp(), released
}

func (t inputTime) tick() int64 {
	return int64(t >> 20)
}
