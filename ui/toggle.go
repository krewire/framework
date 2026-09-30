package ui

import (
	"github.com/krewire/forge/theme"
)

// ThemeModeVarsCSS declares the --show-sun/--show-moon variables that drive
// which icon the theme toggle displays per mode. Include it once, globally
// (unscoped), before ThemeToggleCSS.
const ThemeModeVarsCSS = theme.ThemeModeVarsCSS

// ThemeToggleCSS styles the theme-toggle button rendered by Theme.Button. It
// relies on the palette custom properties (--neutral, --ink, --primary) and
// the mode variables from ThemeModeVarsCSS. Include both globally.
const ThemeToggleCSS = theme.ThemeToggleCSS
