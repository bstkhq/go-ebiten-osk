package osk

// StillInput is an input whose clock stands still. A test that updates a keyboard by hand,
// outside a game's ticks, reads it through one: on the game's clock, which runs on between
// two lines of a test, the input would stop hearing the keyboard.
func StillInput() *Input {
	input := NewInput()
	input.tick = func() int64 { return 0 }

	return input
}
