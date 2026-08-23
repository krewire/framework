package web

import (
	"fmt"
	"net/http"
	"strings"
)

// Controller is a cohesive set of routes. Implement Register and hand the
// controller to Router.Register to mount it.
type Controller interface {
	// Register mounts the controller's routes on r.
	Register(r *Router)
}

// Register mounts c's routes onto the router.
func (r *Router) Register(c Controller) {
	c.Register(r)
}

// RouteBuilder fluently configures a single route before registration.
// Obtain one with Router.Route; finish with Handle.
type RouteBuilder struct {
	router  *Router
	method  string
	pattern string
	name    string
	mws     []Middleware
}

// Route starts a fluent route declaration:
//
//	r.Route(http.MethodGet, "/users/{id}").Name("users.show").Handle(h)
func (r *Router) Route(method, pattern string) *RouteBuilder {
	return &RouteBuilder{router: r, method: method, pattern: pattern}
}

// GET starts a fluent GET route declaration.
func (r *Router) GET(pattern string) *RouteBuilder {
	return r.Route(http.MethodGet, pattern)
}

// POST starts a fluent POST route declaration.
func (r *Router) POST(pattern string) *RouteBuilder {
	return r.Route(http.MethodPost, pattern)
}

// PUT starts a fluent PUT route declaration.
func (r *Router) PUT(pattern string) *RouteBuilder {
	return r.Route(http.MethodPut, pattern)
}

// PATCH starts a fluent PATCH route declaration.
func (r *Router) PATCH(pattern string) *RouteBuilder {
	return r.Route(http.MethodPatch, pattern)
}

// DELETE starts a fluent DELETE route declaration.
func (r *Router) DELETE(pattern string) *RouteBuilder {
	return r.Route(http.MethodDelete, pattern)
}

// Name tags the route for reverse lookup via Router.URL. Reusing a name
// overwrites the previous mapping (last wins).
func (rb *RouteBuilder) Name(name string) *RouteBuilder {
	rb.name = name
	return rb
}

// Use attaches middleware scoped to this single route, applied after any
// group middleware and before the handler.
func (rb *RouteBuilder) Use(mws ...Middleware) *RouteBuilder {
	rb.mws = append(rb.mws, mws...)
	return rb
}

// Handle registers the route with its final handler. Route-level Use
// middleware runs after any group middleware, immediately around the handler.
func (rb *RouteBuilder) Handle(h HandlerFunc) {
	rb.router.Handle(rb.method, rb.pattern, applyChain(rb.mws, h))
	if rb.name != "" {
		root := rb.router.root()
		if root.named == nil {
			root.named = map[string][]segment{}
		}
		root.named[rb.name] = parsePattern(joinPattern(rb.router.base, rb.pattern))
		rb.router.root().dirty = true
	}
}

// PathParams supplies values for {param} segments in reverse URL generation.
type PathParams map[string]string

// URL rebuilds the path of a named route, substituting params. Unknown names
// and missing or empty parameter values return an error.
func (r *Router) URL(name string, params PathParams) (string, error) {
	segs, ok := r.root().named[name]
	if !ok {
		return "", fmt.Errorf("web: unknown route name %q", name)
	}
	var b strings.Builder
	for _, seg := range segs {
		b.WriteByte('/')
		if !seg.hasParam {
			b.WriteString(seg.text)
			continue
		}
		v, ok := params[seg.param]
		if !ok || v == "" || strings.ContainsAny(v, "/?#") {
			return "", fmt.Errorf("web: route %q missing valid value for param %q", name, seg.param)
		}
		b.WriteString(v)
	}
	out := b.String()
	if out == "" {
		out = "/"
	}
	return out, nil
}
