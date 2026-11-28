package osk

type fullLocale struct {
	letterRows, symbolRows, accentRows [3]string
	space, shift, erase                string
	actionLabels                       [ActionSend + 1]string
}

func fullEnUS(kind Kind, action FieldAction) Layout {
	return fullLocale{
		letterRows:   [3]string{"qwertyuiop", "asdfghjkl", "zxcvbnm"},
		symbolRows:   [3]string{"1234567890", ",-_'?!@:/&", "`=[]\\;()\""},
		space:        "Space",
		shift:        "Shift",
		erase:        "Delete",
		actionLabels: [...]string{"Enter", "Next", "Done", "Go", "Search", "Send"},
	}.layout(positionsEnUS(), kind, action)
}

func fullEsES(kind Kind, action FieldAction) Layout {
	return fullLocale{
		letterRows:   [3]string{"qwertyuiop", "asdfghjklñ", "zxcvbnm"},
		symbolRows:   [3]string{"1234567890", ",-_'¿?¡!@+", "`´:;()/º<"},
		accentRows:   [3]string{"áéí", "óú", "üç"},
		space:        "Espacio",
		shift:        "Mayús",
		erase:        "Borrar",
		actionLabels: [...]string{"Intro", "Siguiente", "Listo", "Ir", "Buscar", "Enviar"},
	}.layout(positionsEsES(), kind, action)
}

func (locale fullLocale) layout(positions Positions, kind Kind, action FieldAction) Layout {
	erase := Key{Do: Erase(), Label: locale.erase, Icon: IconErase, Weight: 1.5}
	submit := Key{Do: Submit(action), Label: locale.actionLabels[action], Weight: 2}
	if kind == KindNumber {
		return numberPad(positions, erase, submit)
	}
	shift := Key{Do: Shift(), Label: locale.shift, Icon: IconShift, Weight: 1.5}
	space := Key{Do: Type(" "), Label: locale.space, Weight: 5}
	dot := Key{Do: Type(".")}
	toLetters, toDigits := layerSwitch(letters, "ABC"), layerSwitch(digits, "123")
	withAccents := locale.accentRows[0] != ""
	controls := make([]Key, 0, 4)
	if withAccents {
		toAccents := layerSwitch(accents, "ÁÉÍ")
		space.Weight -= toAccents.weight()
		controls = append(controls, toAccents)
	}
	letterControls := append([]Key{toDigits}, controls...)
	emailSpace := space
	if kind == KindEmail {
		letterControls = append(letterControls, Key{Do: Type("@")})
		emailSpace.Weight--
	}
	letterControls = append(letterControls, emailSpace, dot, submit)
	symbolControls := append([]Key{toLetters}, controls...)
	symbolControls = append(symbolControls, space, dot, submit)
	// The symbols row uses a one-unit erase key to keep its nine symbols aligned
	// with the ten units of the other rows.
	symbolErase := erase
	symbolErase.Weight = 1
	layers := []Layer{
		fullLayer(letters, locale.letterRows, letterControls, []Key{shift}, erase),
		fullLayer(digits, locale.symbolRows, symbolControls, nil, symbolErase),
	}
	if withAccents {
		layers = append(layers, fullLayer(accents, locale.accentRows, []Key{toLetters, toDigits, space, dot, submit}, []Key{shift}, erase))
	}
	return Layout{Layers: layers, Positions: positions}
}

func numberPad(positions Positions, erase, submit Key) Layout {
	erase.Weight, submit.Weight = 1, 1
	return Layout{
		Positions: positions,
		Layers: []Layer{compactLayer(digits, [3]string{"123", "456", "789"},
			erase, Key{Do: Type("0")}, submit)},
	}
}

func fullLayer(name string, rows [3]string, bottom, thirdStart []Key, erase Key) Layer {
	third := append(thirdStart, characters(rows[2])...)
	return Layer{Name: name, Rows: [][]Key{
		characters(rows[0]), characters(rows[1]), append(third, erase), bottom,
	}}
}
