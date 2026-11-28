package osk

import (
	"image"
	"time"
)

const variantDelay = 500 * time.Millisecond

// A variantHold owns a deferred letter until its pointer lifts.
type variantHold struct {
	id    PointerID
	key   *keyState
	at    image.Point
	ready int64
}

// AppendVariants appends the open picker's keys in draw order. Draw them over
// the layer from [Keyboard.AppendKeys]. They may extend above [Keyboard.Rect].
func (k *Keyboard) AppendVariants(keys []PlacedKey) []PlacedKey {
	if k.picker == nil {
		return keys
	}
	hover := variantAt(k.picker, k.picker.at)
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
		if k.holdsVariant(key) {
			return // Another pointer already owns this letter.
		}
		if !landed.lifted {
			k.pending = &variantHold{
				id: pointer.ID, key: key, at: image.Pt(pointer.X, pointer.Y),
				ready: k.updated + int64(durationTicks(variantDelay)),
			}
			return
		}
	}
	k.press(key)
	if landed.lifted {
		k.release(key)
		landed.key = nil
	}
}

// The next key commits a pending tap before it acts, preserving the order
// and Shift state of overlapping taps. An open picker keeps its pointer.
func (k *Keyboard) rolloverVariant(key *keyState) {
	if key != nil && k.pending != nil && key != k.pending.key {
		k.finishVariant(k.pending)
	}
}

func (k *Keyboard) variantForPointer(id PointerID) *variantHold {
	if k.pending != nil && k.pending.id == id {
		return k.pending
	}
	if k.picker != nil && k.picker.id == id {
		return k.picker
	}
	return nil
}

func (k *Keyboard) holdsVariant(key *keyState) bool {
	return k.pending != nil && k.pending.key == key || k.picker != nil && k.picker.key == key
}

func (k *Keyboard) advanceVariants(listening bool) {
	if !listening {
		k.cancelVariants()
		return
	}
	k.sampleVariant(k.picker)
	if k.sampleVariant(k.pending) {
		k.advancePending()
	}
}

func (k *Keyboard) sampleVariant(hold *variantHold) bool {
	if hold == nil {
		return false
	}
	pointer, found := findPointer(k.down, hold.id)
	if found && !pointer.Landed {
		hold.at = image.Pt(pointer.X, pointer.Y)
		return true
	}
	k.finishVariant(hold)
	return false
}

func (k *Keyboard) advancePending() {
	hold := k.pending
	if !hold.at.In(hold.key.Rect) {
		k.pending = nil
		return
	}
	if k.updated >= hold.ready && k.picker == nil {
		k.placeVariants(hold.key)
		k.pending, k.picker = nil, hold
	}
}

func (k *Keyboard) finishVariant(hold *variantHold) {
	key := hold.key
	if hold == k.picker {
		key = variantAt(hold, hold.at)
		k.picker = nil
	} else {
		k.pending = nil
		if !hold.at.In(key.Rect) {
			return
		}
	}
	k.press(key)
	k.release(key)
}

func (k *Keyboard) cancelVariants() {
	k.pending, k.picker = nil, nil
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

func variantAt(hold *variantHold, at image.Point) *keyState {
	for i := range hold.key.variants {
		key := &hold.key.variants[i]
		if at.In(key.Rect) {
			return key
		}
	}
	return nil
}
