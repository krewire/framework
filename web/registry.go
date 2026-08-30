package web

import (
	"net/http"
	"strings"
)

// Registrar is the manual-registry counterpart to Controller.
type Registrar interface {
	RegisterRoutes(r *Router)
}

type registrarFunc func(*Router)

func (fn registrarFunc) RegisterRoutes(r *Router) { fn(r) }

// AsRegistrar wraps fn as a Registrar.
func AsRegistrar(fn func(*Router)) Registrar { return registrarFunc(fn) }

func (r *Router) RegisterRegistrar(reg Registrar) { reg.RegisterRoutes(r) }

func (r *Router) MountRegistrar(reg Registrar) { reg.RegisterRoutes(r) }

// Module is a composable, prefix-scoped bundle of routes for a DDD bounded context.
type Module struct {
	Prefix     string
	Middleware []Middleware
	Register   func(*Router)
}

func (m Module) RegisterRoutes(r *Router) {
	g := r.Group(m.Prefix, m.Middleware...)
	if m.Register != nil {
		m.Register(g)
	}
}

func (r *Router) RegisterModule(m Module) { m.RegisterRoutes(r) }

// ResourceController holds REST handlers. Supports full 7-action shape plus BC aliases.
// Inspired ergonomics: Index/Create/Store/Show/Edit/Update/Destroy, with legacy Delete alias and Create fallback to Store.
type ResourceController struct {
	Index   HandlerFunc // GET /
	Create  HandlerFunc // GET /create
	Store   HandlerFunc // POST /
	Show    HandlerFunc // GET /{id}
	Edit    HandlerFunc // GET /{id}/edit
	Update  HandlerFunc // PUT/PATCH /{id}
	Destroy HandlerFunc // DELETE /{id}
	Delete  HandlerFunc // alias for Destroy (BC)
}

func (c ResourceController) storeHandler() HandlerFunc {
	if c.Store != nil {
		return c.Store
	}
	return c.Create
}

func (c ResourceController) destroyHandler() HandlerFunc {
	if c.Destroy != nil {
		return c.Destroy
	}
	return c.Delete
}

// Resource registers the 7 conventional routes with canonical names.
func (r *Router) Resource(path string, c ResourceController) *Router {
	return r.ResourceFiltered(path, c, ResourceOptions{})
}

// ApiResource registers the 5 API routes (skips create/edit).
func (r *Router) ApiResource(path string, c ResourceController) *Router {
	return r.ApiResourceFiltered(path, c, ResourceOptions{})
}

// ResourceOptions filters resource actions.
type ResourceOptions struct {
	Only   []string
	Except []string
}

// ResourceFiltered registers Resource with Only/Except filtering.
func (r *Router) ResourceFiltered(path string, c ResourceController, opts ResourceOptions) *Router {
	baseName := resourceBaseName(joinPattern(r.base, path))
	actions := map[string]bool{
		"index":   shouldInclude("index", opts),
		"create":  shouldInclude("create", opts),
		"store":   shouldInclude("store", opts),
		"show":    shouldInclude("show", opts),
		"edit":    shouldInclude("edit", opts),
		"update":  shouldInclude("update", opts),
		"destroy": shouldInclude("destroy", opts),
	}
	if actions["index"] && c.Index != nil {
		r.Route(http.MethodGet, path).Name(baseName + ".index").Handle(c.Index)
	}
	if actions["create"] && c.Create != nil {
		r.Route(http.MethodGet, path+"/create").Name(baseName + ".create").Handle(c.Create)
	}
	if actions["store"] {
		if h := c.storeHandler(); h != nil {
			r.Route(http.MethodPost, path).Name(baseName + ".store").Handle(h)
		}
	}
	if actions["show"] && c.Show != nil {
		r.Route(http.MethodGet, path+"/{id}").Name(baseName + ".show").Handle(c.Show)
	}
	if actions["edit"] && c.Edit != nil {
		r.Route(http.MethodGet, path+"/{id}/edit").Name(baseName + ".edit").Handle(c.Edit)
	}
	if actions["update"] && c.Update != nil {
		r.Route(http.MethodPut, path+"/{id}").Name(baseName + ".update").Handle(c.Update)
		r.Route(http.MethodPatch, path+"/{id}").Name(baseName + ".update").Handle(c.Update)
	}
	if actions["destroy"] {
		if h := c.destroyHandler(); h != nil {
			r.Route(http.MethodDelete, path+"/{id}").Name(baseName + ".destroy").Handle(h)
		}
	}
	return r
}

// ApiResourceFiltered registers API resource with filtering (skips create/edit by default filtered).
func (r *Router) ApiResourceFiltered(path string, c ResourceController, opts ResourceOptions) *Router {
	if len(opts.Only) == 0 && len(opts.Except) == 0 {
		opts = ResourceOptions{Except: []string{"create", "edit"}}
	} else {
		// ensure create/edit excluded unless explicitly in Only
		if len(opts.Only) > 0 {
			// Only filtering will handle; if Only includes create/edit they will be included, otherwise not
		} else {
			// Except case: ensure create/edit added to except if not already
			opts.Except = appendUnique(opts.Except, "create", "edit")
		}
	}
	return r.ResourceFiltered(path, c, opts)
}

func appendUnique(base []string, vals ...string) []string {
	have := map[string]bool{}
	for _, v := range base {
		have[v] = true
	}
	for _, v := range vals {
		if !have[v] {
			base = append(base, v)
			have[v] = true
		}
	}
	return base
}

func shouldInclude(action string, opts ResourceOptions) bool {
	if len(opts.Only) > 0 {
		for _, o := range opts.Only {
			if o == action {
				return true
			}
		}
		return false
	}
	for _, e := range opts.Except {
		if e == action {
			return false
		}
	}
	return true
}

func resourceBaseName(path string) string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "resource"
	}
	return strings.ReplaceAll(trimmed, "/", ".")
}
