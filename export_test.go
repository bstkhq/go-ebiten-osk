package osk

// SteppedInput is an input on a clock the test moves: next starts the next tick. A test that
// updates a keyboard by hand, outside a game's ticks, reads it through one: an input answers
// for a keyboard in the tick it was updated in, and the game's own clock runs on between two
// lines of a test.
func SteppedInput() (input *Input, next func()) {
	input = NewInput()

	now := int64(0)
	input.tick = func() int64 { return now }

	return input, func() { now++ }
}
