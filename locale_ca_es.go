package osk

// CaES selects the Catalan layout for Spain.
const CaES Locale = "ca-ES"

var catalan = fullLocale{
	letterRows:   [3]string{"qwertyuiop", "asdfghjklñç", "zxcvbnm"},
	symbolRows:   [3]string{"1234567890", ",-_'¿?¡!@+", "`´:;()/º<"},
	accentRows:   [3]string{"àèé", "íïòó", "úü·"},
	accentLabel:  "ÀÈÉ",
	space:        "Espai",
	shift:        "Majús",
	erase:        "Esborra",
	actionLabels: [...]string{"Retorn", "Següent", "Fet", "Vés", "Cerca", "Envia"},
}

// xkeyboard-config's es(cat) shares the Spanish base level.
func compactCaES() Layout {
	return catalan.compact(positionsEsES())
}

func fullCaES(kind Kind, action FieldAction) Layout {
	return catalan.layout(positionsEsES(), kind, action)
}
