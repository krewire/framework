package web

// BaseController is an optional embed for API controllers that need shared middleware and helpers.
//
// Inspired ergonomics (no framework magic): a controller owns its route-scoped middleware and helpers,
// while the router owns registration:
//
//	type UserController struct { BaseController }
//	func (c *UserController) Register(r *Router) {
//	    r.Get("/{id}", c.Show)
//	}
//	ctrl := &UserController{}
//	ctrl.Use(AuthMiddleware)
//	r.Group("/users", ctrl.Middleware()...).Register(ctrl)
//
// It composes with generic handlers H[Q] and Response helpers.
//
// Migration map (fluent grouping inspired by popular frameworks):
//
//	apiResource  -> r.ApiResource("/users", ctrl)
//	middleware   -> r.Middleware(auth).Group(...)
//	prefix+name  -> r.Prefix("/admin").Name("admin.")
//	fallback     -> r.Fallback(handler)
//	redirect     -> r.Redirect("/old","/new",301)
//	any/match    -> r.Any / r.Match
type BaseController struct {
	middlewares []Middleware
}

// Use appends middleware to the controller's scoped stack.
func (b *BaseController) Use(mw ...Middleware) { b.middlewares = append(b.middlewares, mw...) }

// Middleware returns the controller's scoped middleware (for Group).
func (b *BaseController) Middleware() []Middleware {
	return append([]Middleware(nil), b.middlewares...)
}

// Helpers for use inside H[Q] handlers.

// OK returns a 200 JSON response helper (use inside H[Q]: return ctrl.OK(data)).
func (b *BaseController) OK(v any) *Response { return Respond().JSON(v) }

// Created returns a 201 JSON response.
func (b *BaseController) Created(v any) *Response { return Created(v) }

// NoContent returns a 204 response.
func (b *BaseController) NoContent() *Response { return NoContent() }

// Validate wraps Request.Validate for controller helpers.
func (b *BaseController) Validate(r *Request, dst any) error { return r.Validate(dst) }

// BadRequest returns a 400 error helper.
func (b *BaseController) BadRequest(msg string) error { return BadRequest(msg) }

// NotFoundErr returns a 404 error helper (named to avoid clash with web.NotFound type).
func (b *BaseController) NotFoundErr(msg string) error { return NotFound(msg) }

// Forbidden returns a 403 error.
func (b *BaseController) Forbidden(msg string) error { return Forbidden(msg) }
