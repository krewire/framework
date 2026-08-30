package web

import (
	"fmt"
	"io/fs"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
)

// Params holds the path variables extracted from a route pattern.
type Params map[string]string

// HandlerFunc handles an HTTP request with the route's parameters.
type HandlerFunc func(http.ResponseWriter, *http.Request, Params)

// Route pairs an HTTP method and path pattern with a handler.
type Route struct {
	method      string
	pattern     string
	name        string
	segments    []segment
	handle      HandlerFunc
	constraints map[string]string
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

	namePrefix string
	where      map[string]string

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

// Options registers a handler for OPTIONS requests on a path pattern.
func (r *Router) Options(pattern string, h HandlerFunc) {
	r.Handle(http.MethodOptions, pattern, h)
}

// Head registers a handler for HEAD requests on a path pattern.
func (r *Router) Head(pattern string, h HandlerFunc) {
	r.Handle(http.MethodHead, pattern, h)
}

// Any registers a handler for all common HTTP methods on a path pattern.
func (r *Router) Any(pattern string, h HandlerFunc) {
	for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodHead} {
		r.Handle(m, pattern, h)
	}
}

// Match registers a handler for the given HTTP methods on a path pattern.
func (r *Router) Match(methods []string, pattern string, h HandlerFunc) {
	for _, m := range methods {
		if m == "" {
			continue
		}
		r.Handle(strings.ToUpper(strings.TrimSpace(m)), pattern, h)
	}
}

// Fallback registers a catch-all handler for unmatched routes.
func (r *Router) Fallback(h HandlerFunc) {
	r.root().NotFound = h
	r.root().dirty = true
}

// Redirect registers a GET route that redirects to `to`.
func (r *Router) Redirect(from, to string, code ...int) {
	c := http.StatusFound
	if len(code) > 0 {
		c = code[0]
	}
	r.Get(from, func(w http.ResponseWriter, req *http.Request, _ Params) {
		http.Redirect(w, req, to, c)
	})
}

// PermanentRedirect registers a GET route that permanently redirects (301) to `to`.
func (r *Router) PermanentRedirect(from, to string) {
	r.Redirect(from, to, http.StatusMovedPermanently)
}

// Prefix is an alias for Group(prefix) for fluent grouping (inspired ergonomics).
func (r *Router) Prefix(prefix string) *Router { return r.Group(prefix) }

// Name returns a derived router that prefixes all route names registered through it.
func (r *Router) Name(prefix string) *Router {
	return r.namePrefixRouter(prefix)
}

// As is an alias for Name (fluent `As("api.")`).
func (r *Router) As(prefix string) *Router { return r.Name(prefix) }

func (r *Router) namePrefixRouter(prefix string) *Router {
	np := r.namePrefix + prefix
	whereCopy := cloneWhere(r.where)
	chainCopy := append([]Middleware(nil), r.chain...)
	return &Router{base: r.base, parent: r.root(), chain: chainCopy, namePrefix: np, where: whereCopy}
}

// Middleware returns a derived router that adds middleware to the current chain (fluent).
func (r *Router) Middleware(mws ...Middleware) *Router { return r.Group("", mws...) }

// Where adds a param constraint to the group (introspection only; not enforced in match yet).
func (r *Router) Where(param, constraint string) *Router {
	nw := cloneWhere(r.where)
	if nw == nil {
		nw = map[string]string{}
	}
	nw[param] = constraint
	chainCopy := append([]Middleware(nil), r.chain...)
	return &Router{base: r.base, parent: r.root(), chain: chainCopy, namePrefix: r.namePrefix, where: nw}
}

func cloneWhere(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
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
	resolved := joinPattern(r.base, pattern)
	route := &Route{method: method, pattern: resolved, segments: parsePattern(resolved), handle: h}
	if r.where != nil {
		route.constraints = cloneWhere(r.where)
	}
	root.routes = append(root.routes, route)
	root.dirty = true
}

// MustHandle is Handle that panics on duplicate method+pattern.
func (r *Router) MustHandle(method, pattern string, h HandlerFunc) {
	if r.HasRoute(method, pattern) {
		panic("web: duplicate route " + method + " " + joinPattern(r.base, pattern))
	}
	r.Handle(method, pattern, h)
}

// Mount is shorthand for Group(prefix,mws...).Register-style mounting:
// r.Mount("/api", func(api *Router){ api.Get("/users", h) })
func (r *Router) Mount(prefix string, fn func(*Router), mws ...Middleware) *Router {
	g := r.Group(prefix, mws...)
	if fn != nil {
		fn(g)
	}
	return g
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
	return &Router{base: "/" + strings.Trim(base, "/"), parent: r.root(), chain: chain, namePrefix: r.namePrefix, where: cloneWhere(r.where)}
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

// RouteInfo describes one registered route for introspection.
type RouteInfo struct {
	Method      string
	Pattern     string
	Name        string
	Static      bool
	Constraints map[string]string
}

// Routes returns a snapshot of all registered routes and statics, in
// registration order (statics appended after routes).
func (r *Router) Routes() []RouteInfo {
	root := r.root()
	out := make([]RouteInfo, 0, len(root.routes)+len(root.statics))
	for _, rt := range root.routes {
		out = append(out, RouteInfo{Method: rt.method, Pattern: rt.pattern, Name: rt.name, Constraints: cloneWhere(rt.constraints)})
	}
	for _, st := range root.statics {
		out = append(out, RouteInfo{Method: "STATIC", Pattern: st.prefix, Static: true})
	}
	return out
}

// HasRoute reports whether method+pattern is already registered. Pattern is
// normalized via joinPattern against the receiver's base when the receiver is
// a Group; call on the root for absolute checks.
func (r *Router) HasRoute(method, pattern string) bool {
	root := r.root()
	normalized := pattern
	if !strings.HasPrefix(normalized, "/") && normalized != "" {
		normalized = "/" + normalized
	}
	resolved := normalized
	if r.base != "" {
		if strings.HasPrefix(normalized, r.base) && (len(normalized) == len(r.base) || normalized[len(r.base)] == '/') {
			resolved = normalized
		} else {
			resolved = joinPattern(r.base, normalized)
		}
	}
	for _, rt := range root.routes {
		if rt.method == method && rt.pattern == resolved {
			return true
		}
	}
	return false
}

// RouteExists is an alias for HasRoute.
func (r *Router) RouteExists(method, pattern string) bool { return r.HasRoute(method, pattern) }

// CheckDuplicates returns every route occurrence after its first appearance
// (method+pattern). Empty means the table is conflict-free.
func (r *Router) CheckDuplicates() []RouteInfo {
	root := r.root()
	seen := map[string]bool{}
	var dups []RouteInfo
	for _, rt := range root.routes {
		key := rt.method + " " + rt.pattern
		if seen[key] {
			dups = append(dups, RouteInfo{Method: rt.method, Pattern: rt.pattern, Name: rt.name})
			continue
		}
		seen[key] = true
	}
	return dups
}

// Validate reports the first duplicate route as an error, or nil if none.
func (r *Router) Validate() error {
	if dups := r.CheckDuplicates(); len(dups) > 0 {
		d := dups[0]
		return fmt.Errorf("web: duplicate route %s %s", d.Method, d.Pattern)
	}
	return nil
}

// DebugString returns a deterministic, human-readable route manifest.
func (r *Router) DebugString() string {
	infos := r.Routes()
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].Method != infos[j].Method {
			return infos[i].Method < infos[j].Method
		}
		return infos[i].Pattern < infos[j].Pattern
	})
	var b strings.Builder
	for _, ri := range infos {
		if ri.Static {
			fmt.Fprintf(&b, "%-7s %s\n", ri.Method, ri.Pattern)
			continue
		}
		if ri.Name != "" {
			if len(ri.Constraints) > 0 {
				fmt.Fprintf(&b, "%-7s %-30s -> %s %v\n", ri.Method, ri.Pattern, ri.Name, ri.Constraints)
			} else {
				fmt.Fprintf(&b, "%-7s %-30s -> %s\n", ri.Method, ri.Pattern, ri.Name)
			}
		} else {
			if len(ri.Constraints) > 0 {
				fmt.Fprintf(&b, "%-7s %s %v\n", ri.Method, ri.Pattern, ri.Constraints)
			} else {
				fmt.Fprintf(&b, "%-7s %s\n", ri.Method, ri.Pattern)
			}
		}
	}
	return b.String()
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
