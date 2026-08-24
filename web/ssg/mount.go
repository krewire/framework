package ssg

import (
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/krewire/framework/runtime/mount"
)

// renderMount renders a component server-side and wraps it with hydration
// markers (KWF-T4X9P FRK-WASM-040). Template usage:
//
//	{{mount "Counter" "load" .}}
//
// The inner HTML is the fully scoped component output, so CSS and semantics
// are identical to a plain {{component}} call; the wrapper only adds the
// data-kiw-mount contract the client runtime scans. Props JSON strips
// framework control keys via componentProps, matching registry decoding.
func (s *Site) renderMount(name, hydrate string, data any) (template.HTML, error) {
	h, err := mount.ParseHydrate(hydrate)
	if err != nil {
		return "", fmt.Errorf("ssg: mount %q: %w", name, err)
	}
	inner, err := s.renderComponent(name, data)
	if err != nil {
		return "", fmt.Errorf("ssg: mount %q: %w", name, err)
	}
	propsJSON := ""
	if p := componentProps(data); p != nil {
		b, mErr := json.Marshal(p)
		if mErr != nil {
			return "", fmt.Errorf("ssg: mount %q props: %w", name, mErr)
		}
		propsJSON = string(b)
	}
	return template.HTML(mount.Wrap(name, h, propsJSON, string(inner))), nil
}
