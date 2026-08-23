package ssg

import (
	"html/template"
	"sync"

	"github.com/krewire/framework/ui"
)

// Component is a reusable piece of UI. Body is the html/template source;
// Style is CSS scoped to the component's rendered root element.
type Component struct {
	Name  string
	Body  string
	Style string
}

// Layout wraps page content in a shared shell. Body receives a LayoutData
// value (fields Title, Content, Data). Style is scoped to the layout's root
// element.
type Layout struct {
	Name  string
	Body  string
	Style string
}

// LayoutData is the value passed to a layout template.
type LayoutData struct {
	// Title is the page title.
	Title string
	// Content is the rendered page component HTML.
	Content template.HTML
	// Data is the page's data value.
	Data any
}

// Page is one output page of the site.
type Page struct {
	// Path is the output URL: "/" for the root, "/about" for a clean URL, or
	// "/about.html" for an explicit filename.
	Path string
	// Title is passed to the layout.
	Title string
	// Layout names the Layout wrapping this page.
	Layout string
	// Root names the root component of the page.
	Root string
	// Data is passed to the root component.
	Data any
	// Props, when the Site emits props, is serialized to a data-props
	// attribute on the page's root element for client-side mounting.
	Props any
	// Scripts are <script src> URLs injected before the closing body tag,
	// empty by default.
	Scripts []string
}

// Site builds a static website from components, layouts, pages, and assets.
type Site struct {
	funcs     template.FuncMap
	layouts   map[string]*Layout
	comps     map[string]*Component
	pages     []*Page
	assets    map[string]string
	reg       *ui.Registry
	emitProps bool

	set *template.Template
	mu  sync.Mutex
	// used records every component and layout whose styles were referenced
	// during a build.
	used map[string]bool
}

// New returns an empty Site.
func New() *Site {
	return &Site{
		layouts: map[string]*Layout{},
		comps:   map[string]*Component{},
		assets:  map[string]string{},
		used:    map[string]bool{},
	}
}
