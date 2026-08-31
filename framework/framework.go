// Package framework is the entry point for the Krewire meta-framework.
//
// Rather than being built as a single, self-contained ecosystem, Krewire
// composes modules sourced across multiple ecosystems and repositories,
// orchestrating them into one cohesive, high-level framework.

package framework

import (
	"fmt"

	fw "github.com/krewire/framework"
)

// Name is the canonical display name of the Krewire framework.
const Name = "Krewire Framework"

// Version is the semantic version of the framework, aliased from the module
// root (framework/version.go) so the framework has a single source of truth
// for its version. kiw release edits only the root declaration.
var Version = fw.Version

// Banner returns the framework banner identifying the current framework version.
func Banner() string {
	return fmt.Sprintf("%s v%s", Name, Version.String())
}
