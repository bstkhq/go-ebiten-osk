package osk

import "github.com/hajimehoshi/ebiten/v2"

// WithEraseRepeat sets how many game ticks after the initial erase press the first repeat
// comes, and the ticks between repeats. The defaults are 30 and 6 (500 ms and 100 ms at
// 60 ticks per second). Either value at or below zero disables repeating.
func WithEraseRepeat(delay, interval int) Option {
	return func(o *options) {
		o.repeat = eraseRepeat{delay: int64(delay), interval: int64(interval)}
	}
}

// eraseRepeat tracks the hold shared by every pressed Erase key on a keyboard.
type eraseRepeat struct {
	delay, interval int64
	next            int64
	held            int
}

func (r *eraseRepeat) press(now int64) {
	if r.held == 0 {
		r.next = now + r.delay
	}
	r.held++
}

func (r *eraseRepeat) due(now int64) bool {
	if r.held == 0 || r.delay <= 0 || r.interval <= 0 || now != r.next {
		return false
	}
	r.next += r.interval
	return true
}

func (k *Keyboard) repeatErase() {
	if k.repeat.due(k.updated) && k.held[ebiten.KeyBackspace] == k.repeat.held {
		k.positionUp(ebiten.KeyBackspace)
		k.positionDown(ebiten.KeyBackspace)
	}
}
