package component

import (
	"fmt"
	"sort"
	"sync"
)

var (
	regMu    sync.RWMutex
	registry = map[string]Factory{}
)

// Register adds a component factory under the given name — the same key the
// SSG emits as data-kiw-component and hydration uses to instantiate client
// components. Re-registering an existing name is an error.
func Register(name string, f Factory) error {
	regMu.Lock()
	defer regMu.Unlock()
	if _, dup := registry[name]; dup {
		return fmt.Errorf("component %q already registered", name)
	}
	if f == nil {
		return fmt.Errorf("component %q: nil factory", name)
	}
	registry[name] = f
	return nil
}

// MustRegister registers and panics on error; intended for package init.
func MustRegister(name string, f Factory) {
	if err := Register(name, f); err != nil {
		panic(err)
	}
}

// Lookup returns the factory registered under name.
func Lookup(name string) (Factory, error) {
	regMu.RLock()
	defer regMu.RUnlock()
	f, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("component %q not registered", name)
	}
	return f, nil
}

// Names returns a sorted snapshot of registered component names.
func Names() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	out := make([]string, 0, len(registry))
	for n := range registry {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
