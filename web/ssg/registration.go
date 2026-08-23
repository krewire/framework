package ssg

import (
	"html/template"
	"sort"

	"github.com/krewire/framework/ui"
)

// Funcs registers template functions available to components and layouts.
func (s *Site) Funcs(f template.FuncMap) *Site {
	if s.funcs == nil {
		s.funcs = template.FuncMap{}
	}
	for name, fn := range f {
		s.funcs[name] = fn
	}
	return s
}

// Component registers a component by name.
func (s *Site) Component(c Component) *Site {
	s.comps[c.Name] = &c
	return s
}

// Layout registers a layout by name.
func (s *Site) Layout(l Layout) *Site {
	s.layouts[l.Name] = &l
	return s
}

// Page registers an output page.
func (s *Site) Page(p Page) *Site {
	s.pages = append(s.pages, &p)
	return s
}

// Asset registers a static file by its output path, e.g. "favicon.ico" or
// "assets/theme.css".
func (s *Site) Asset(name, body string) *Site {
	s.assets[name] = body
	return s
}

// Assets returns the names of all registered assets in sorted order.
func (s *Site) Assets() []string {
	names := make([]string, 0, len(s.assets))
	for name := range s.assets {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// AssetBody returns the content of a registered asset.
func (s *Site) AssetBody(name string) (string, bool) {
	body, ok := s.assets[name]
	return body, ok
}

// Registry sets the component registry used to resolve component names that
// don't correspond to template bodies. Resolution order: registered Go-native
// components first, then template components defined on the site.
func (s *Site) Registry(reg *ui.Registry) *Site {
	s.reg = reg
	return s
}

// EmitProps enables serializing a page's Props to a data-props attribute on
// the page's root element, providing a stable mount point for client-side
// frameworks. Off by default; pages with a nil Props are left untouched.
func (s *Site) EmitProps() *Site {
	s.emitProps = true
	return s
}
