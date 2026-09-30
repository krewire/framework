// Package ui provides the Krewire UI Framework.
// The theming system is powered by github.com/krewire/forge/theme.
package ui

import (
	"github.com/krewire/forge/theme"
)

// Color is a CSS color value, e.g. "#0b58a1", "rgb(11 88 161)", or
// "oklch(0.5 0.2 250)".
type Color = theme.Color

// Palette is a named color schema for a single mode (light or dark).
type Palette = theme.Palette

// DefaultLightPalette is the palette applied when no light override is set.
var DefaultLightPalette = theme.DefaultLightPalette

// DefaultDarkPalette is the palette applied when no dark override is set.
var DefaultDarkPalette = theme.DefaultDarkPalette
