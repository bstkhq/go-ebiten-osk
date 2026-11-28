package osk

import (
	"image"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

const variantDelay = 500 * time.Millisecond

// A variantHold owns a deferred letter until its pointer lifts.
type variantHold struct {
	id        PointerID
	key       *keyState
	at        image.Point
	ready     int64
	cancelled bool
}

// AppendVariants appends the open picker's keys in draw order. Draw them over
// the layer from [Keyboard.AppendKeys]. They may extend above [Keyboard.Rect].
func (k *Keyboard) AppendVariants(keys []PlacedKey) []PlacedKey {
	if k.picker == nil {
		return keys
	}
	hover := choiceAt(k.picker, k.picker.at)
	for i := range k.picker.key.variants {
		key := &k.picker.key.variants[i]
		placed := k.placed(key)
		if key == hover {
			placed.State = StatePressed
		}
		keys = append(keys, placed)
	}
	return keys
}

func (k *Keyboard) pressLanding(pointer Pointer, landed *landing) {
	key := landed.key
	if key != nil && len(key.variants) > 0 {
		landed.key = nil
		if key.State == StatePressed {
			return // Another pointer already owns this letter.
		}
		if !landed.lifted {
			hold := &variantHold{
				id: pointer.ID, key: key, at: image.Pt(pointer.X, pointer.Y),
				ready: k.updated + int64(durationTicks(variantDelay)),
			}
			key.State = StatePressed
			landed.variant = hold
			k.variantHolds = append(k.variantHolds, hold)
			return
		}
	}
	k.press(key)
	if landed.lifted {
		k.release(key)
		landed.key = nil
	}
}

func (k *Keyboard) advanceVariants(listening bool) {
	if !listening {
		k.cancelVariants()
		return
	}
	kept := k.variantHolds[:0]
	for _, hold := range k.variantHolds {
		pointer, found := findPointer(k.down, hold.id)
		if found && !pointer.Landed {
			k.moveVariant(hold, image.Pt(pointer.X, pointer.Y))
			kept = append(kept, hold)
			continue
		}
		k.releaseVariant(hold, found)
	}
	clear(k.variantHolds[len(kept):])
	k.variantHolds = kept
}

func (k *Keyboard) moveVariant(hold *variantHold, at image.Point) {
	hold.at = at
	if hold.cancelled || hold == k.picker {
		return
	}
	if !at.In(hold.key.Rect) {
		hold.cancelled = true
		return
	}
	if k.updated >= hold.ready && k.picker == nil {
		k.placeVariants(hold.key)
		k.picker = hold
	}
}

func (k *Keyboard) releaseVariant(hold *variantHold, relanded bool) {
	// Stock mouse coordinates remain available in the release tick.
	if !relanded && hold.id == Mouse {
		if _, stock := k.pointers.(*screen); stock {
			hold.at = image.Pt(ebiten.CursorPosition())
		}
	}
	k.finishVariant(hold)
}

func (k *Keyboard) finishVariant(hold *variantHold) {
	hold.key.State = StateIdle
	if hold.cancelled {
		return
	}
	key := hold.key
	if hold == k.picker {
		key = choiceAt(hold, hold.at)
		k.picker = nil
	} else if !hold.at.In(key.Rect) {
		return
	}
	k.press(key)
	k.release(key)
}

func (k *Keyboard) cancelVariants() {
	for _, hold := range k.variantHolds {
		hold.key.State = StateIdle
	}
	clear(k.variantHolds)
	k.variantHolds = k.variantHolds[:0]
	k.picker = nil
}

func (k *Keyboard) placeVariants(key *keyState) {
	count := len(key.variants)
	width := min(key.Rect.Dx()*count, k.rect.Dx())
	height := max(1, key.Rect.Dy()*3/4)
	left := min(key.Rect.Min.X, k.rect.Max.X-width)
	top := key.Rect.Min.Y - height
	if top < 0 {
		top = key.Rect.Max.Y
	}
	for i := range key.variants {
		key.variants[i].Rect = image.Rect(left+i*width/count, top, left+(i+1)*width/count, top+height)
	}
}

func choiceAt(hold *variantHold, at image.Point) *keyState {
	for i := range hold.key.variants {
		key := &hold.key.variants[i]
		if at.In(key.Rect) {
			return key
		}
	}
	return nil
}
