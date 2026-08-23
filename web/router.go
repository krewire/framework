package web

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
)

// Params holds the path variables extracted from a route pattern.
type Params map[string]string

// HandlerFunc handles an HTTP request with the route's parameters.
type HandlerFunc func(http.ResponseWriter, *http.Request, Params)

// Route pairs an HTTP method and path pattern with a handler.
type Route struct {
	method   string
	segments []segment
	handle   HandlerFunc
}

type segment struct {
	text     string
	param    string
	hasParam bool
}

// Router registers routes and dispatches requests.
type Router struct {
	routes  []*Route
	statics []staticRoute
	// NotFound, when set, handles unmatched routes.
	NotFound HandlerFunc
	mws      []Middleware

	base   string
	parent *Router
	// chain holds group-scoped middleware applied to routes registered
	// through this router (nested groups inherit).
	chain []Middleware
	// named maps route names to their segments for reverse URL generation
	// (root only).
	named map[string][]segment

	built http.Handler
	dirty bool
}

type staticRoute struct {
	prefix string
	dir    string
	fsys   fs.FS
}

// NewRouter returns an empty Router.
func NewRouter() *Router {
	return &Router{}
}

// Get registers a handler for GET requests on a path pattern.
func (r *Router) Get(pattern string, h HandlerFunc) {
	r.Handle(http.MethodGet, pattern, h)
}

// Post registers a handler for POST requests on a path pattern.
func (r *Router) Post(pattern string, h HandlerFunc) {
	r.Handle(http.MethodPost, pattern, h)
}

// Put registers a handler for PUT requests on a path pattern.
func (r *Router) Put(pattern string, h HandlerFunc) {
	r.Handle(http.MethodPut, pattern, h)
}

// Delete registers a handler for DELETE requests on a path pattern.
func (r *Router) Delete(pattern string, h HandlerFunc) {
	r.Handle(http.MethodDelete, pattern, h)
}

// Patch registers a handler for PATCH requests on a path pattern.
func (r *Router) Patch(pattern string, h HandlerFunc) {
	r.Handle(http.MethodPatch, pattern, h)
}

// Handle registers a handler for the given HTTP method and path pattern.
// Path segments of the form "{name}" become parameters. Any middleware on the
// router's group chain wraps the handler with matched params injected.
func (r *Router) Handle(method, pattern string, h HandlerFunc) {
	if h == nil {
		h = func(http.ResponseWriter, *http.Request, Params) {}
	}
	h = applyChain(r.chain, h)
	root := r.root()
	root.routes = append(root.routes, &Route{method: method, segments: parsePattern(joinPattern(r.base, pattern)), handle: h})
	root.dirty = true
}

// applyChain folds route-level middleware around h, outermost first, adapting
// standard http.Handler middleware so handlers keep receiving Params.
func applyChain(chain []Middleware, h HandlerFunc) HandlerFunc {
	for i := len(chain) - 1; i >= 0; i-- {
		mw := chain[i]
		next := h
		h = func(w http.ResponseWriter, req *http.Request, p Params) {
			var inner HandlerFunc = next
			stub := http.HandlerFunc(func(rw http.ResponseWriter, rr *http.Request) {
				inner(rw, rr, p)
			})
			mw(stub).ServeHTTP(w, req)
		}
	}
	return h
}

// Use registers middleware applied to every route in registration order.
func (r *Router) Use(mw ...Middleware) {
	root := r.root()
	root.mws = append(root.mws, mw...)
	root.dirty = true
}

// Group returns a scoped router sharing the parent's middleware. Routes and
// static dirs registered on the group are mounted under prefix. Optional mws
// scope middleware to exactly this group's routes; nested groups inherit.
// Use (on any router) remains global.
func (r *Router) Group(prefix string, mws ...Middleware) *Router {
	base := strings.Trim(r.base, "/")
	if p := strings.Trim(prefix, "/"); p != "" {
		base += "/" + p
	}
	chain := make([]Middleware, 0, len(r.chain)+len(mws))
	chain = append(chain, r.chain...)
	chain = append(chain, mws...)
	return &Router{base: "/" + strings.Trim(base, "/"), parent: r.root(), chain: chain}
}

// Static serves files from dir under the URL prefix.
func (r *Router) Static(prefix, dir string) {
	root := r.root()
	root.statics = append(root.statics, staticRoute{prefix: joinPattern(r.base, prefix), dir: dir})
	root.dirty = true
}

// StaticFS serves files from fsys (an embed-compatible fs.FS) under the URL
// prefix.
func (r *Router) StaticFS(prefix string, fsys fs.FS) {
	root := r.root()
	root.statics = append(root.statics, staticRoute{prefix: joinPattern(r.base, prefix), fsys: fsys})
	root.dirty = true
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler().ServeHTTP(w, req)
}

func (r *Router) root() *Router {
	if r.parent != nil {
		return r.parent
	}
	return r
}

func (r *Router) handler() http.Handler {
	if r.built != nil && !r.dirty {
		return r.built
	}
	var h http.Handler = http.HandlerFunc(r.serve)
	for i := len(r.mws) - 1; i >= 0; i-- {
		h = r.mws[i](h)
	}
	r.built = h
	r.dirty = false
	return h
}

func (r *Router) serve(w http.ResponseWriter, req *http.Request) {
	for _, rt := range r.routes {
		if rt.method != req.Method {
			continue
		}
		if params, ok := rt.match(req.URL.Path); ok {
			rt.handle(w, req, params)
			return
		}
	}
	for _, st := range r.statics {
		if !strings.HasPrefix(req.URL.Path, st.prefix) {
			continue
		}
		if st.fsys != nil {
			http.StripPrefix(st.prefix, http.FileServer(http.FS(st.fsys))).ServeHTTP(w, req)
			return
		}
		rel := strings.TrimPrefix(req.URL.Path, st.prefix)
		file := filepath.Join(st.dir, filepath.Clean("/"+rel))
		http.ServeFile(w, req, file)
		return
	}
	if r.NotFound != nil {
		r.NotFound(w, req, nil)
		return
	}
	http.NotFound(w, req)
}

// joinPattern mounts pattern under base, preserving a leading slash when the
// base has one (e.g. base "/api/v1" + pattern "/users" -> "/api/v1/users").
func joinPattern(base, pattern string) string {
	trimmed := strings.Trim(pattern, "/")
	if base == "" {
		return "/" + trimmed
	}
	return strings.TrimRight(base, "/") + "/" + trimmed
}

// match reports whether path matches the route pattern and returns extracted
// parameters.
func (rt *Route) match(path string) (Params, bool) {
	parts := splitPath(path)
	if len(parts) != len(rt.segments) {
		return nil, false
	}
	params := make(Params, len(rt.segments))
	for i, seg := range rt.segments {
		if seg.hasParam {
			if parts[i] == "" {
				return nil, false
			}
			params[seg.param] = parts[i]
			continue
		}
		if seg.text != parts[i] {
			return nil, false
		}
	}
	return params, true
}

func parsePattern(pattern string) []segment {
	segs := make([]segment, 0, 8)
	for _, part := range splitPath(pattern) {
		if len(part) > 2 && strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			segs = append(segs, segment{param: part[1 : len(part)-1], hasParam: true})
			continue
		}
		segs = append(segs, segment{text: part})
	}
	return segs
}

// splitPath splits a URL path into non-empty segments.
func splitPath(p string) []string {
	return strings.FieldsFunc(p, func(r rune) bool { return r == '/' })
}
