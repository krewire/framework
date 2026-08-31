// Package gateway provides an API gateway (KWF-L5H2F FRK-SVC-020/021/022/023).
// It owns a route table, proxies requests to services discovered via the
// service registry, and applies middleware (logging, tracing, CORS, auth,
// rate limiting, circuit breaker). Missing upstreams return 502 with a
// structured Problem JSON response.
package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/krewire/framework/service/registry"
)

// Route maps a path/method pair to a target service (FRK-SVC-020).
type Route struct {
	Path       string
	Method     string
	Service    string
	Middleware []string
	RateLimit  int
	Auth       bool
}

// Gateway is the API gateway (FRK-SVC-020).
type Gateway struct {
	mu       sync.RWMutex
	routes   []Route
	registry registry.Registry
	mws      map[string]Middleware
	proxy    *httputil.ReverseProxy
}

// Middleware is a function that wraps an http.Handler (FRK-SVC-021).
type Middleware func(http.Handler) http.Handler

// Option configures a Gateway.
type Option func(*Gateway)

// WithRegistry sets the service registry for discovery.
func WithRegistry(r registry.Registry) Option {
	return func(g *Gateway) { g.registry = r }
}

// WithMiddleware registers named middleware.
func WithMiddleware(name string, mw Middleware) Option {
	return func(g *Gateway) { g.mws[name] = mw }
}

// New creates a gateway with the given options.
func New(opts ...Option) *Gateway {
	g := &Gateway{
		mws: map[string]Middleware{},
	}
	for _, o := range opts {
		o(g)
	}
	return g
}

// AddRoute registers a route (FRK-SVC-020).
func (g *Gateway) AddRoute(r Route) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.routes = append(g.routes, r)
}

// Reload atomically replaces the route table (FRK-SVC-022).
func (g *Gateway) Reload(routes []Route) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.routes = routes
}

// ServeHTTP implements http.Handler (FRK-SVC-020).
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mu.RLock()
	routes := g.routes
	g.mu.RUnlock()

	var matched *Route
	for i := range routes {
		if routes[i].Path == r.URL.Path && (routes[i].Method == "" || routes[i].Method == r.Method) {
			matched = &routes[i]
			break
		}
	}
	if matched == nil {
		writeProblem(w, http.StatusNotFound, "no route for "+r.URL.Path)
		return
	}

	// Build middleware chain
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.proxyTo(w, r, matched.Service)
	})
	for i := len(matched.Middleware) - 1; i >= 0; i-- {
		if mw, ok := g.mws[matched.Middleware[i]]; ok {
			handler = mw(handler)
		}
	}
	handler.ServeHTTP(w, r)
}

// proxyTo discovers the service and proxies the request (FRK-SVC-020).
func (g *Gateway) proxyTo(w http.ResponseWriter, r *http.Request, serviceName string) {
	if g.registry == nil {
		writeProblem(w, http.StatusBadGateway, "no registry configured")
		return
	}
	eps, err := g.registry.Discover(r.Context(), serviceName)
	if err != nil || len(eps) == 0 {
		writeProblem(w, http.StatusBadGateway, "service "+serviceName+" unavailable")
		return
	}
	target := eps[0].Address
	u, err := url.Parse(schemeHostURL(target))
	if err != nil {
		writeProblem(w, http.StatusBadGateway, "invalid upstream address")
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		writeProblem(w, http.StatusBadGateway, "upstream error: "+err.Error())
	}
	proxy.ServeHTTP(w, r)
}

// Problem is a RFC 7807 Problem Details response (FRK-SVC-023).
type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

// writeProblem writes a RFC 7807 Problem JSON response (FRK-SVC-023).
func writeProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	})
}

// schemeHostURL parses a host address into a URL for reverse proxy.
func schemeHostURL(addr string) string {
	if len(addr) > 7 && addr[:7] == "http://" {
		return addr
	}
	if len(addr) > 8 && addr[:8] == "https://" {
		return addr
	}
	return "http://" + addr
}
