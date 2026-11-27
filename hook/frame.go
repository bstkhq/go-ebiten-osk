package hook

import (
	osk "github.com/bstkhq/go-ebiten-osk"
	"github.com/hajimehoshi/ebiten/v2"
)

type frame struct {
	tick  int64
	keys  [ebiten.KeyMax + 1]keyUpdate
	runes []rune
}

type keyUpdate struct {
	holders   int
	restarted int
	pressed   bool
}

func (f *frame) start(tick int64) {
	if f.tick != tick {
		f.tick = tick
		clear(f.keys[:])
		f.runes = f.runes[:0]
	}
}

func (h *Hook) capture(held [ebiten.KeyMax + 1]bool, events []osk.Event) {
	f := &h.pending
	f.start(ebiten.Tick())
	var up, down [ebiten.KeyMax + 1]bool
	for _, event := range events {
		switch event.Kind {
		case osk.KeyUp:
			up[event.Key] = true
		case osk.KeyDown:
			down[event.Key] = true
		case osk.Text:
			f.runes = append(f.runes, []rune(event.Text)...)
		}
	}
	for key, held := range held {
		f.keys[physicalKey(ebiten.Key(key))].add(held, up[key], down[key])
	}
}

func (k *keyUpdate) add(held, up, down bool) {
	k.pressed = k.pressed || down
	if held {
		k.holders++
		if up && down {
			k.restarted++
		}
	}
}

// Ebitengine only reads combined modifiers from their left and right slots.
func physicalKey(key ebiten.Key) ebiten.Key {
	switch key {
	case ebiten.KeyAlt:
		return ebiten.KeyAltLeft
	case ebiten.KeyControl:
		return ebiten.KeyControlLeft
	case ebiten.KeyShift:
		return ebiten.KeyShiftLeft
	case ebiten.KeyMeta:
		return ebiten.KeyMetaLeft
	default:
		return key
	}
}
