package web

import "github.com/krewire/libs/sec"

// CORSOptions configures Cross-Origin Resource Sharing.
type CORSOptions = sec.CORSOptions

var (
	// WithOrigins appends allowed origins to CORSOptions.
	WithOrigins = sec.WithOrigins
	// WithMethods appends allowed HTTP methods to CORSOptions.
	WithMethods = sec.WithMethods
	// WithHeaders appends allowed headers to CORSOptions.
	WithHeaders = sec.WithHeaders
	// WithExposeHeaders appends exposed response headers to CORSOptions.
	WithExposeHeaders = sec.WithExposeHeaders
	// WithCredentials sets AllowCredentials in CORSOptions.
	WithCredentials = sec.WithCredentials
	// WithMaxAge sets preflight cache duration in seconds.
	WithMaxAge = sec.WithMaxAge
)

// CORS returns middleware applying Cross-Origin Resource Sharing rules.
// Delegates to sec.CORS as the foundational security layer.
func CORS(opts ...func(*CORSOptions)) Middleware {
	return Middleware(sec.CORS(opts...))
}
