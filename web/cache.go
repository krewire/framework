package web

import "net/http"

// CacheControl returns middleware setting Cache-Control verbatim.
func CacheControl(value string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", value)
			next.ServeHTTP(w, r)
		})
	}
}

// NoStore forbids caching of private/dynamic responses.
func NoStore() Middleware {
	return CacheControl("no-store, private")
}

// MaxAge allows caching for n seconds; public also permits shared caches.
func MaxAge(seconds int, public bool) Middleware {
	scope := "private"
	if public {
		scope = "public"
	}
	return CacheControl(scope + ", max-age=" + strconvItoa(seconds))
}

// Immutable serves fingerprinted assets with long-lived caching.
func Immutable(maxAgeSeconds int) Middleware {
	return CacheControl("public, max-age=" + strconvItoa(maxAgeSeconds) + ", immutable")
}
