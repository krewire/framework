package web

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/krewire/framework/ui"
	"github.com/krewire/framework/web/ssg"
)

// App assembles a monolith: it composes routes, middleware, pages, static
// assets, theming, and the shared component registry into one HTTP listener
// that serves HTML pages (SSR), JSON APIs (KWF-H3QD8), and embedded assets.
//
// Pages render through the same ssg render path used for static export, so
// live output and exported output are byte-identical for identical input.
//
// File responsibilities: app.go (aggregate and fluent registration),
// serve.go (HTTP serving), site.go (ssg.Site assembly and export).
type App struct {
	router *Router

	funcs     template.FuncMap
	comps     []ssg.Component
	layouts   []ssg.Layout
	assets    map[string]string
	registry  *ui.Registry
	theme     *ui.Theme
	emitProps bool

	pages   []*PageSpec
	statics []staticFS

	renderSite *ssg.Site
	dirty      bool
}

type staticFS struct {
	prefix string
	fsys   fs.FS
}

// PageSpec describes one page of an App: a root component rendered inside a
// layout. A page is static when Data is fixed, or dynamic when DataFunc
// computes per-request data.
type PageSpec struct {
	// Path is the served URL and export path ("/" for the root).
	Path string
	// Title is passed to the layout.
	Title string
	// Layout names the layout wrapping this page.
	Layout string
	// Root names the root component of the page, resolved through the registry.
	Root string
	// Data is the fixed page data for static pages.
	Data any
	// DataFunc, when set, computes the page data per request. Pages with a
	// DataFunc are dynamic and skipped by Export.
	DataFunc func(*http.Request) (any, error)
	// EmitProps serializes the page data to a data-props attribute on the
	// root element for client-side mounting. Off by default.
	EmitProps bool
	// Scripts are <script src> URLs injected before </body>, empty by default.
	Scripts []string
}

// NewApp returns an App pre-loaded with the default component registry, a
// trusted-HTML template helper, and the default middleware stack: recovery
// outermost, then access logging (KWL-P8W2N KWF-HTTPV-006). Explicit Use
// calls compose outside these.
func NewApp() *App {
	a := &App{
		router:   NewRouter(),
		assets:   map[string]string{},
		registry: ui.Default(),
		funcs:    template.FuncMap{},
	}
	a.router.Use(RecoverMiddleware(nil), AccessLogMiddleware(nil))
	a.funcs = template.FuncMap{
		"html": func(v any) template.HTML {
			return template.HTML(fmt.Sprint(v))
		},
		"themeHead": func() template.HTML {
			if a.theme == nil {
				return template.HTML("")
			}
			return a.theme.Script()
		},
		"themeButton": func() template.HTML {
			if a.theme == nil {
				return template.HTML("")
			}
			return a.theme.Button()
		},
	}
	a.dirty = true
	return a
}

// Router exposes the underlying router for API routes and middleware.
func (a *App) Router() *Router { return a.router }

// Use registers middleware applied to every route.
func (a *App) Use(mw ...Middleware) { a.router.Use(mw...) }

// Method registers an API or HTML handler on the router.
func (a *App) Method(method, pattern string, h HandlerFunc) {
	a.router.Handle(method, pattern, h)
}

// Funcs registers template functions available to components and layouts.
func (a *App) Funcs(f template.FuncMap) *App {
	for name, fn := range f {
		if a.funcs == nil {
			a.funcs = template.FuncMap{}
		}
		a.funcs[name] = fn
	}
	a.dirty = true
	return a
}

// Component registers a template component by name.
func (a *App) Component(c ssg.Component) *App {
	a.comps = append(a.comps, c)
	a.dirty = true
	return a
}

// Layout registers a layout by name.
func (a *App) Layout(l ssg.Layout) *App {
	a.layouts = append(a.layouts, l)
	a.dirty = true
	return a
}

// Asset registers a static file by its output path (e.g. "favicon.ico").
func (a *App) Asset(name, body string) *App {
	a.assets[name] = body
	a.dirty = true
	return a
}

// Theme attaches the theming system; its CSS is collected into the site's
// assets.
func (a *App) Theme(t *ui.Theme) *App {
	a.theme = t
	a.dirty = true
	return a
}

// Registry overrides the component registry used to resolve page roots.
func (a *App) Registry(r *ui.Registry) *App {
	a.registry = r
	a.dirty = true
	return a
}

// EmitProps globally enables data-props emission for pages that request it.
func (a *App) EmitProps() *App {
	a.emitProps = true
	a.dirty = true
	return a
}

// Page registers a page.
func (a *App) Page(p PageSpec) *App {
	a.pages = append(a.pages, &p)
	a.dirty = true
	return a
}

// Static mounts an embed-compatible fs.FS (e.g. an //go:embed value) under
// prefix, serving its files untouched.
func (a *App) Static(prefix string, fsys fs.FS) *App {
	a.statics = append(a.statics, staticFS{prefix: prefix, fsys: fsys})
	return a
}

func pageData(spec *PageSpec, req *http.Request) (any, error) {
	if spec.DataFunc == nil {
		return spec.Data, nil
	}
	return spec.DataFunc(req)
}

func propsFor(spec *PageSpec, data any) any {
	if !spec.EmitProps {
		return nil
	}
	return data
}
