package plugin

// Plugin is a site build extension. It is detected by presence of its
// config files at the project root (e.g., tailwind.config.js) and runs
// as part of `krewire build` before the final site is emitted. Tailwind
// is the first example; the interface is intentionally minimal so other
// PostCSS plugins can follow the same pattern without becoming part of the
// core framework.
type Plugin interface {
	Name() string
	// Detect reports whether the plugin should run for the project at root.
	Detect(root string) bool
	// Build runs the plugin, writing its output into outDir (the site output
	// directory, e.g., site/). It may read config and source files from root.
	Build(root, outDir string) error
}

// Registry holds all known plugins. Plugins self-register via init().
var Registry []Plugin

func Register(p Plugin) {
	Registry = append(Registry, p)
}
