package osk

import (
	"slices"
	"strings"
	"unicode/utf8"
)

const fullControlWidth = 1.5

type fullLocale struct {
	letterRows, symbolRows, accentRows [3]string
	space, shift, erase                string
	accentLabel                        string
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
		accentLabel:  "ÁÉÍ",
		space:        "Espacio",
		shift:        "Mayús",
		erase:        "Borrar",
		actionLabels: [...]string{"Intro", "Siguiente", "Listo", "Ir", "Buscar", "Enviar"},
	}.layout(positionsEsES(), kind, action)
}

func (locale fullLocale) layout(positions Positions, kind Kind, action FieldAction) Layout {
	erase := Key{Do: Erase(), Label: locale.erase, Icon: IconErase, Weight: fullControlWidth}
	submit := Key{Do: Submit(action), Label: locale.actionLabels[action], Weight: fullControlWidth}
	if kind == KindNumber {
		return numberPad(positions, erase, submit)
	}
	if action == ActionEnter {
		submit.Icon = IconAction
	}
	return Layout{Layers: locale.fullLayers(kind, erase, submit), Positions: positions}
}

func (locale fullLocale) fullLayers(kind Kind, erase, submit Key) []Layer {
	shift := Key{Do: Shift(), Label: locale.shift, Icon: IconShift, Weight: fullControlWidth}
	tail := []Key{{Do: Type(" "), Label: locale.space, Weight: 4}, {Do: Type(".")}, submit}
	toLetters, toDigits := layerSwitch(letters, "ABC"), layerSwitch(digits, "123")
	toLetters.Weight, toDigits.Weight = fullControlWidth, fullControlWidth
	letterControls, symbolControls := []Key{toDigits}, []Key{toLetters}
	withAccents := locale.accentRows[0] != ""
	if withAccents {
		toAccents := layerSwitch(accents, locale.accentLabel)
		toAccents.Weight = 1
		letterControls = append(letterControls, toAccents)
		symbolControls = append(symbolControls, toAccents)
	}
	context := ","
	if kind == KindEmail {
		context = "@"
	}
	letterTail := append([]Key{{Do: Type(context)}}, tail...)

	grid := locale.grid()
	letterBottom := grid.row(letterControls, nil, letterTail)
	symbolRows, extra := grid.splitSymbols(locale.symbolRows)
	symbolBottom := grid.row(symbolControls, nil, append(extra, tail...))
	layers := []Layer{
		grid.layer(letters, locale.letterRows, letterBottom, []Key{shift}, erase),
		grid.layer(digits, symbolRows, symbolBottom, nil, erase),
	}
	if withAccents {
		accentBottom := grid.row([]Key{toLetters, toDigits}, nil, tail)
		layers = append(layers, grid.layer(accents, locale.accentRows, accentBottom, []Key{shift}, erase))
	}
	return layers
}

func (locale fullLocale) compact(positions Positions) Layout {
	space := Key{Do: Type(" "), Label: locale.space, Weight: 5}
	erase := Key{Do: Erase(), Label: locale.erase, Weight: 3}
	toLetters, toDigits := layerSwitch(letters, "ABC"), layerSwitch(digits, "123")
	var accentsControl []Key
	withAccents := locale.accentRows[0] != ""
	if withAccents {
		toAccents := layerSwitch(accents, locale.accentLabel)
		space.Weight -= toAccents.weight()
		accentsControl = []Key{toAccents}
	}
	letterControls := append(append([]Key{toDigits}, accentsControl...), space, erase)
	symbolControls := append(append([]Key{toLetters}, accentsControl...), space, erase)
	locale.symbolRows[1] = "." + locale.symbolRows[1]
	for row := range locale.letterRows {
		locale.letterRows[row] = strings.ToUpper(locale.letterRows[row])
		locale.accentRows[row] = strings.ToUpper(locale.accentRows[row])
	}
	layers := []Layer{
		compactLayer(letters, locale.letterRows, letterControls...),
		compactLayer(digits, locale.symbolRows, symbolControls...),
	}
	if withAccents {
		layers = append(layers, compactLayer(accents, locale.accentRows, toLetters, toDigits, space, erase))
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

// fullGrid keeps a common slot pitch across the layers. The letter span
// anchors the edge keys while longer rows can use the full width.
type fullGrid struct {
	width, letterSpan float64
}

func (locale fullLocale) grid() fullGrid {
	slots := func(row string) float64 { return float64(utf8.RuneCountInString(row)) }
	letterSpan := max(stockRowWidth, slots(locale.letterRows[0]), slots(locale.letterRows[2])+2*fullControlWidth)
	width := max(letterSpan, slots(locale.letterRows[1]), slots(locale.symbolRows[1]))
	return fullGrid{width: width, letterSpan: letterSpan}
}

// splitSymbols moves any symbols that would crowd erase to the bottom row.
func (grid fullGrid) splitSymbols(rows [3]string) ([3]string, []Key) {
	third := []rune(rows[2])
	split := min(len(third), int(grid.letterSpan-fullControlWidth))
	rows[2] = string(third[:split])
	return rows, characters(string(third[split:]), grid.width)
}

type fullRow struct {
	keys []Key
	gaps []float64
}

func (grid fullGrid) layer(name string, rows [3]string, bottom fullRow, thirdStart []Key, erase Key) Layer {
	third := grid.row(thirdStart, characters(rows[2], grid.width), []Key{erase})
	return Layer{
		Name:    name,
		Rows:    [][]Key{characters(rows[0], grid.width), characters(rows[1], grid.width), third.keys, bottom.keys},
		RowGaps: [][]float64{nil, nil, third.gaps, bottom.gaps},
	}
}

// row centers the middle keys between the keys anchored at each edge.
func (grid fullGrid) row(left, middle, right []Key) fullRow {
	keys := slices.Concat(left, middle, right)
	var used float64
	for _, key := range keys {
		used += key.weight()
	}
	span := max(grid.letterSpan, used)
	margin, gap := (grid.width-span)/2, (span-used)/2
	gaps := make([]float64, len(keys)+1)
	gaps[0], gaps[len(keys)] = margin, margin
	gaps[len(left)] += gap
	gaps[len(left)+len(middle)] += gap
	return fullRow{keys: keys, gaps: gaps}
}
