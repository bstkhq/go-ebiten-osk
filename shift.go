package osk

func (k *Keyboard) activate(key *keyState) {
	switch key.Do.kind {
	case shifts:
		k.shift = nextShift(k.shift)
	case submits:
		k.events = append(k.events, Event{Kind: ActionPressed, FieldAction: key.Do.field})
	case types:
		text := k.keyText(key)
		k.events = append(k.events, Event{Kind: Text, Text: text})
		if k.shift == StateLatched && text != key.Do.text {
			k.shift = StateIdle
		}
	}
}

func nextShift(state State) State {
	switch state {
	case StateIdle:
		return StateLatched
	case StateLatched:
		return StateLocked
	default:
		return StateIdle
	}
}

func (k *Keyboard) placed(key *keyState) PlacedKey {
	placed := key.PlacedKey
	placed.Do.text = k.keyText(key)
	if key.Do.kind == shifts && k.shift != StateIdle {
		placed.State = k.shift
	}
	return placed
}

func (k *Keyboard) keyText(key *keyState) string {
	if k.shift != StateIdle {
		return key.capital
	}
	return key.Do.text
}
