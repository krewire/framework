package web

import (
	"github.com/krewire/framework/ui"
	"github.com/krewire/framework/web/ssg"
)

// Export writes the App's static pages as a complete website into outDir,
// byte-identical to the ssg build for the same input. Dynamic pages (those
// with a DataFunc) are skipped.
func (a *App) Export(outDir string) ([]string, error) {
	s := a.exportSite()
	return s.Build(outDir)
}

// site builds the shared render site, cached until the App is mutated.
func (a *App) site() *ssg.Site {
	if a.renderSite != nil && !a.dirty {
		return a.renderSite
	}
	a.renderSite = a.buildSite(nil)
	a.dirty = false
	return a.renderSite
}

// exportSite builds a site for static export: dynamic pages are skipped.
func (a *App) exportSite() *ssg.Site {
	var static []PageSpec
	for _, p := range a.pages {
		if p.DataFunc == nil {
			static = append(static, *p)
		}
	}
	return a.buildSite(static)
}

func (a *App) buildSite(pages []PageSpec) *ssg.Site {
	s := ssg.New().Funcs(a.funcs).Registry(a.registry)
	if emit := a.emitEnabled(); emit {
		s.EmitProps()
	}
	for _, c := range a.comps {
		s.Component(c)
	}
	for _, l := range a.layouts {
		s.Layout(l)
	}
	for name, body := range a.assets {
		s.Asset(name, body)
	}
	if a.theme != nil {
		s.Asset("assets/theme.css", ui.ThemeModeVarsCSS+"\n"+ui.ThemeToggleCSS)
	}
	if a.registry != nil {
		s.Asset("assets/ui.css", ui.ComponentsCSS())
	}
	for _, p := range pages {
		s.Page(ssg.Page{
			Path:    p.Path,
			Title:   p.Title,
			Layout:  p.Layout,
			Root:    p.Root,
			Data:    p.Data,
			Props:   propsFor(&p, p.Data),
			Scripts: p.Scripts,
		})
	}
	return s
}

// emitEnabled reports whether any page requests data-props emission, either
// globally at the App level or per page.
func (a *App) emitEnabled() bool {
	if a.emitProps {
		return true
	}
	for _, p := range a.pages {
		if p.EmitProps {
			return true
		}
	}
	return false
}
