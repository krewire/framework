// Package theme defines the application palette and toggle styling
// (FRK-STR-006). Palette default: orange neon light/dark, as the ecosystem
// default.
package theme

import "github.com/krewire/framework/ui"

// New returns the application theme with the default Krewire palette.
func New() *ui.Theme {
	return &ui.Theme{
		Light: ui.Palette{Primary: "#ff6b00"},
		Dark:  ui.Palette{Primary: "#ff8c42"},
	}
}
