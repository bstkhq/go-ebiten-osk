package osk

type fullLocale struct {
	letterRows, symbolRows, accentRows [3]string
	space, shift, erase                string
	actionLabels                       [ActionSend + 1]string
}

func fullEnUS(action FieldAction) Layout {
	return fullLocale{
		letterRows:   [3]string{"qwertyuiop", "asdfghjkl", "zxcvbnm"},
		symbolRows:   [3]string{"1234567890", ",-_'?!@:/&", "`=[]\\;()\""},
		space:        "Space",
		shift:        "Shift",
		erase:        "Delete",
		actionLabels: [...]string{"Enter", "Next", "Done", "Go", "Search", "Send"},
	}.layout(positionsEnUS(), action)
}

func fullEsES(action FieldAction) Layout {
	return fullLocale{
		letterRows:   [3]string{"qwertyuiop", "asdfghjklñ", "zxcvbnm"},
		symbolRows:   [3]string{"1234567890", ",-_'¿?¡!@+", "`´:;()/º<"},
		accentRows:   [3]string{"áéí", "óú", "üç"},
		space:        "Espacio",
		shift:        "Mayús",
		erase:        "Borrar",
		actionLabels: [...]string{"Intro", "Siguiente", "Listo", "Ir", "Buscar", "Enviar"},
	}.layout(positionsEsES(), action)
}

func (locale fullLocale) layout(positions Positions, action FieldAction) Layout {
	shift := Key{Do: Shift(), Label: locale.shift, Icon: IconShift, Weight: 1.5}
	erase := Key{Do: Erase(), Label: locale.erase, Icon: IconErase, Weight: 1.5}
	space := Key{Do: Type(" "), Label: locale.space, Weight: 5}
	dot := Key{Do: Type(".")}
	submit := Key{Do: Submit(action), Label: locale.actionLabels[action], Weight: 2}
	toLetters, toDigits := layerSwitch(letters, "ABC"), layerSwitch(digits, "123")
	withAccents := locale.accentRows[0] != ""
	controls := make([]Key, 0, 4)
	if withAccents {
		toAccents := layerSwitch(accents, "ÁÉÍ")
		space.Weight -= toAccents.weight()
		controls = append(controls, toAccents)
	}
	controls = append(controls, space, dot, submit)
	// The symbols row uses a one-unit erase key to keep its nine symbols aligned
	// with the ten units of the other rows.
	symbolErase := erase
	symbolErase.Weight = 1
	layers := []Layer{
		fullLayer(letters, locale.letterRows, append([]Key{toDigits}, controls...), []Key{shift}, erase),
		fullLayer(digits, locale.symbolRows, append([]Key{toLetters}, controls...), nil, symbolErase),
	}
	if withAccents {
		layers = append(layers, fullLayer(accents, locale.accentRows, []Key{toLetters, toDigits, space, dot, submit}, []Key{shift}, erase))
	}
	return Layout{Layers: layers, Positions: positions}
}

func fullLayer(name string, rows [3]string, bottom, thirdStart []Key, erase Key) Layer {
	third := append(thirdStart, characters(rows[2])...)
	return Layer{Name: name, Rows: [][]Key{
		characters(rows[0]), characters(rows[1]), append(third, erase), bottom,
	}}
}
