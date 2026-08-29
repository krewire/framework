package plugin

import "strings"

// Plugin is a site build extension. It is detected by presence of its
// config files at the project root (e.g., tailwind.config.js) and runs
// as part of `krewire build` before the final site is emitted. Tailwind
// is the first example; the interface is intentionally minimal so other
// PostCSS plugins can follow the same pattern without becoming part of the
// core framework.
//
// The Add/Remove extension makes the plugin system scalable: kiw add/remove
// dispatch through the same registry, so new plugins require zero changes to
// the devtool. Non-plugin packages (npm, Go) are handled by the generic
// resolvers in kiw/internal/packages.
type Plugin interface {
	Name() string
	// Aliases are alternative names for kiw add (e.g., "twcss" for tailwind).
	Aliases() []string
	// Detect reports whether the plugin should run for the project at root.
	Detect(root string) bool
	// Build runs the plugin, writing its output into outDir (the site output
	// directory, e.g., site/). It may read config and source files from root.
	Build(root, outDir string) error
}

// Installer is the optional install/uninstall contract for a plugin. A
// plugin that implements this can be managed via `kiw add/remove name@version`.
// Plugins that do not implement it are still usable via manual config but
// cannot be added/removed automatically.
type Installer interface {
	Plugin
	// Add installs the plugin at version ("" or "latest" means latest).
	Add(root, version string) error
	// Remove uninstalls the plugin.
	Remove(root string) error
}

// Registry holds all known plugins. Plugins self-register via init().
var Registry []Plugin

func Register(p Plugin) {
	Registry = append(Registry, p)
}

// Find returns the plugin whose Name or Aliases match name (case-insensitive).
func Find(name string) Plugin {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, p := range Registry {
		if strings.ToLower(p.Name()) == lower {
			return p
		}
		if aliased, ok := p.(interface{ Aliases() []string }); ok {
			for _, a := range aliased.Aliases() {
				if strings.ToLower(a) == lower {
					return p
				}
			}
		}
		// also allow npm name tailwindcss -> tailwind plugin
		if lower == "tailwindcss" && strings.ToLower(p.Name()) == "tailwind" {
			return p
		}
	}
	return nil
}

// FindInstaller returns the Installer for name if the plugin implements it.
func FindInstaller(name string) Installer {
	p := Find(name)
	if p == nil {
		return nil
	}
	ins, _ := p.(Installer)
	return ins
}
