// Package paint draws an on-screen keyboard from its state and a theme, with go-shapes and etxt.
// Start from [Dark] or [Light], change the values you need, and pass the theme to [New] or
// [Painter.SetTheme]. To draw your own keyboard, read the keyboard's public state directly.
package paint

import (
	"image/color"
	"sync"

	osk "github.com/bstkhq/go-ebiten-osk"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/sfnt"
)

// A Theme is the values that give a keyboard its look. Copy a stock theme and change the
// fields you need. Use colour values in overrides so they are copied with the theme.
type Theme struct {
	Palette Palette
	Roles   [osk.RoleAction + 1]RoleColors
	States  [osk.StateLocked + 1]KeyColors

	// Font draws the labels. Nil uses Go Bold.
	Font *sfnt.Font

	// Sizes are fractions of a row's height. A zero label size hides that size of label.
	// CharacterSize is for one character and WordSize for longer labels.
	CharacterSize float64
	WordSize      float64
	Gap           float64 // space between keys
	Radius        float64 // rounding of the key's corners
	Padding       float64 // space between the face's edge and its label or icon
	LipHeight     float64 // how much of the lower edge shows under the face
	BorderWidth   float64
}

// A Palette sets the panel colour and the base colours of the keys and labels.
// Control includes the space bar and layer switches.
type Palette struct {
	Panel, Key, Control, Action, Label color.RGBA
}

// KeyColors are the colours of a key. In an override, nil inherits that colour and
// [color.Transparent] explicitly sets it to transparent.
type KeyColors struct {
	Face, Lip, Border, Label color.Color
}

// RoleColors overrides colours for one role, for all states or for a specific state.
type RoleColors struct {
	All    KeyColors
	States [osk.StateLocked + 1]KeyColors
}

// Colors resolves a key's colours from the palette, then applies overrides in this order:
// role, state, and the particular role and state together. Unknown roles and states use
// the control and idle colours, respectively.
func (t Theme) Colors(role osk.Role, state osk.State) KeyColors {
	if uint(role) >= uint(len(t.Roles)) {
		role = osk.RoleControl
	}
	if uint(state) >= uint(len(t.States)) {
		state = osk.StateIdle
	}

	base := t.Palette.base(role)
	var contrast uint8
	if 3*int(base.R)+6*int(base.G)+int(base.B) < 5*int(base.A) {
		contrast = base.A
	}
	amount := [...]int{osk.StatePressed: 16, osk.StateLatched: 28, osk.StateLocked: 40}[state]
	colors := KeyColors{
		Face:   shade(base, contrast, amount),
		Lip:    shade(base, 0, 42+amount/2),
		Border: shade(base, contrast, 12+amount),
		Label:  t.Palette.Label,
	}
	colors.apply(t.Roles[role].All)
	colors.apply(t.States[state])
	colors.apply(t.Roles[role].States[state])
	return colors
}

func (p Palette) base(role osk.Role) color.RGBA {
	switch role {
	case osk.RoleCharacter:
		return p.Key
	case osk.RoleAction:
		return p.Action
	default:
		return p.Control
	}
}

// shade blends toward black or white at the same alpha, in premultiplied channels.
func shade(base color.RGBA, target uint8, percent int) color.RGBA {
	blend := func(channel uint8) uint8 {
		return uint8((int(channel)*(100-percent) + int(target)*percent + 50) / 100)
	}
	return color.RGBA{R: blend(base.R), G: blend(base.G), B: blend(base.B), A: base.A}
}

func (c *KeyColors) apply(override KeyColors) {
	if override.Face != nil {
		c.Face = override.Face
	}
	if override.Lip != nil {
		c.Lip = override.Lip
	}
	if override.Border != nil {
		c.Border = override.Border
	}
	if override.Label != nil {
		c.Label = override.Label
	}
}

// Dark is the default theme, with light labels on charcoal keys, in Go Bold.
func Dark() Theme {
	return stock(Palette{Panel: rgb(0x17191d), Key: rgb(0x343840), Control: rgb(0x434956), Action: rgb(0x325289), Label: rgb(0xf1f2f4)})
}

// Light is the stock theme with dark labels on pale keys, in Go Bold.
func Light() Theme {
	return stock(Palette{Panel: rgb(0xd4dae3), Key: rgb(0xfefeff), Control: rgb(0xe5ebf4), Action: rgb(0xaccaf7), Label: rgb(0x192335)})
}

func rgb(hex uint32) color.RGBA {
	return color.RGBA{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 0xff}
}

func stock(palette Palette) Theme {
	return Theme{
		Palette:       palette,
		Font:          goBold(),
		CharacterSize: 0.44,
		WordSize:      0.30,
		Gap:           0.10,
		Radius:        0.12,
		Padding:       0.045,
		LipHeight:     0.07,
		BorderWidth:   0.02,
	}
}

// goBold is Go Bold, parsed once. Every stock theme shares it.
var goBold = sync.OnceValue(parseGoBold)

func parseGoBold() *sfnt.Font {
	font, err := sfnt.Parse(gobold.TTF)
	if err != nil {
		panic("paint: Go Bold, which ships with x/image, does not parse: " + err.Error())
	}

	return font
}
