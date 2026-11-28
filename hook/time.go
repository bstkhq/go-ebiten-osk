package hook

import "github.com/hajimehoshi/ebiten/v2"

// inputTime mirrors ui.InputTime. Its high bits encode the engine tick.
type inputTime int64

const inputTimeSubtickBits = 20

// stamp and stamp()+1 are two ordered timestamps at the end of the current
// tick. The hook writes to the game snapshot after the backend has populated
// it, so it can stamp the snapshot without advancing the backend's event clock.
func stamp() inputTime {
	const first = (1 << inputTimeSubtickBits) - 2
	return inputTime(ebiten.Tick()<<inputTimeSubtickBits | first)
}

func (t inputTime) tick() int64 {
	return int64(t >> inputTimeSubtickBits)
}
