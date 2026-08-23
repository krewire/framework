package ui

// ThemeModeVarsCSS declares the --show-sun/--show-moon variables that drive
// which icon the theme toggle displays per mode. Include it once, globally
// (unscoped), before ThemeToggleCSS.
const ThemeModeVarsCSS = `:root { --show-sun: block; --show-moon: none; }
:root[data-theme="dark"] { --show-sun: none; --show-moon: block; }
@media (prefers-color-scheme: dark) { :root:not([data-theme]) { --show-sun: none; --show-moon: block; } }`

// ThemeToggleCSS styles the theme-toggle button rendered by Theme.Button. It
// relies on the palette custom properties (--neutral, --ink, --primary) and
// the mode variables from ThemeModeVarsCSS. Include both globally.
const ThemeToggleCSS = `.theme-toggle { border: 1px solid var(--neutral); border-radius: 999px; background: transparent; color: var(--ink); padding: .35rem .5rem; line-height: 0; cursor: pointer; transition: border-color .2s ease, color .2s ease; }
.theme-toggle:hover { border-color: var(--primary); color: var(--primary); }
.theme-toggle svg { width: 1.05rem; height: 1.05rem; display: block; }
.theme-toggle .icon-sun { display: var(--show-sun); }
.theme-toggle .icon-moon { display: var(--show-moon); }`
