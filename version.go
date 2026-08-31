package framework

import "github.com/krewire/libs/core"

// Version is the framework module version.
var Version = core.MustParseVersion("0.3.1")

// EcosystemRequires declares the minimum versions of other modules this version is compatible with.
var EcosystemRequires = map[core.ModuleName]core.Version{
	core.ModuleLibs: core.MustParseVersion("0.3.0"),
}
