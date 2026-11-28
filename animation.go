package osk

import (
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

// DefaultAnimationDuration is the time the keyboard takes to show or hide at
// the configured tick rate.
const DefaultAnimationDuration = 200 * time.Millisecond

// WithAnimationDuration sets the length of a show or hide transition. A
// non-positive duration disables animation. New keyboards start fully shown.
func WithAnimationDuration(duration time.Duration) Option {
	return func(o *options) { o.animation = duration }
}

// SetAnimationDuration sets the length of the next transition from an endpoint.
// A transition already in progress keeps its timing, including if reversed.
// A non-positive duration finishes the current transition immediately and
// disables animation.
func (k *Keyboard) SetAnimationDuration(duration time.Duration) {
	k.animation.duration = duration
	if duration <= 0 {
		k.animation.step = k.animation.target
	}
}

// Visibility returns the eased fraction of the keyboard that is on screen.
// It is zero when fully hidden and one when fully shown. A painter can use
// it for both the distance traveled and the opacity.
func (k *Keyboard) Visibility() float64 {
	remaining := float64(k.animation.ticks-k.animation.step) / float64(k.animation.ticks)
	return 1 - remaining*remaining
}

// A transition walks the same steps in either direction. The endpoints are
// zero and ticks, so changing direction preserves both position and easing.
type animation struct {
	duration time.Duration
	ticks    int
	step     int
	target   int
}

func (a *animation) aim(show bool) {
	if show == (a.target != 0) {
		return
	}
	if a.step == a.target {
		a.ticks = durationTicks(a.duration)
		if !show {
			a.step = a.ticks
		}
	}
	a.target = 0
	if show {
		a.target = a.ticks
	}
	if a.duration <= 0 {
		a.step = a.target
	}
}

func durationTicks(duration time.Duration) int {
	tps := ebiten.TPS()
	if tps <= 0 {
		// An update per frame has no configured tick rate.
		tps = 60
	}
	ticks := math.Ceil(float64(duration) * float64(tps) / float64(time.Second))
	if ticks >= float64(math.MaxInt) {
		return math.MaxInt
	}
	return max(1, int(ticks))
}

func (a *animation) advance() {
	switch {
	case a.step < a.target:
		a.step++
	case a.step > a.target:
		a.step--
	}
}

func (a *animation) acceptsKeys() bool {
	return a.target == a.ticks && a.step == a.ticks
}
