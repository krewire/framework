package i18n

import (
	"net/http"
	"strconv"
	"strings"
)

// MiddlewareOption configures the i18n HTTP middleware.
type MiddlewareOption func(*middlewareConfig)

type middlewareConfig struct {
	queryParam   string
	cookieName   string
	headerName   string
	enableQuery  bool
	enableCookie bool
	enableHeader bool
}

// WithQueryParam sets the query parameter name for locale detection (default: "lang").
func WithQueryParam(name string) MiddlewareOption {
	return func(c *middlewareConfig) {
		c.queryParam = name
		c.enableQuery = name != ""
	}
}

// WithCookieName sets the cookie name for locale detection (default: "lang").
func WithCookieName(name string) MiddlewareOption {
	return func(c *middlewareConfig) {
		c.cookieName = name
		c.enableCookie = name != ""
	}
}

// WithHeaderName sets the header name for locale detection (default: "Accept-Language").
func WithHeaderName(name string) MiddlewareOption {
	return func(c *middlewareConfig) {
		c.headerName = name
		c.enableHeader = name != ""
	}
}

// Middleware creates an HTTP middleware that extracts the user's preferred locale
// and injects it into request context.
// Detection order:
// 1. URL Query parameter (?lang=...)
// 2. Cookie (lang=...)
// 3. Accept-Language header
// 4. Bundle default locale
func Middleware(bundle *Bundle, opts ...MiddlewareOption) func(http.Handler) http.Handler {
	cfg := &middlewareConfig{
		queryParam:   "lang",
		cookieName:   "lang",
		headerName:   "Accept-Language",
		enableQuery:  true,
		enableCookie: true,
		enableHeader: true,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			locale := detectLocale(r, bundle, cfg)
			ctx := WithLocale(r.Context(), locale)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func detectLocale(r *http.Request, bundle *Bundle, cfg *middlewareConfig) string {
	// 1. Query param
	if cfg.enableQuery && cfg.queryParam != "" {
		if val := r.URL.Query().Get(cfg.queryParam); val != "" {
			cleaned := strings.ToLower(strings.TrimSpace(val))
			if bundle == nil || bundle.HasLocale(cleaned) {
				return cleaned
			}
		}
	}

	// 2. Cookie
	if cfg.enableCookie && cfg.cookieName != "" {
		if cookie, err := r.Cookie(cfg.cookieName); err == nil && cookie.Value != "" {
			cleaned := strings.ToLower(strings.TrimSpace(cookie.Value))
			if bundle == nil || bundle.HasLocale(cleaned) {
				return cleaned
			}
		}
	}

	// 3. Accept-Language header
	if cfg.enableHeader && cfg.headerName != "" {
		if headerVal := r.Header.Get(cfg.headerName); headerVal != "" {
			if matched := matchAcceptLanguage(headerVal, bundle); matched != "" {
				return matched
			}
		}
	}

	// 4. Default locale
	if bundle != nil {
		return bundle.DefaultLocale()
	}
	return "en"
}

// matchAcceptLanguage parses an Accept-Language header value and finds the best matching locale.
// e.g. "id-ID,id;q=0.9,en-US;q=0.8,en;q=0.7"
func matchAcceptLanguage(header string, bundle *Bundle) string {
	if header == "" || bundle == nil {
		return ""
	}

	type langEntry struct {
		tag     string
		quality float64
	}

	var entries []langEntry
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		subParts := strings.Split(part, ";")
		tag := strings.ToLower(strings.TrimSpace(subParts[0]))
		quality := 1.0

		for _, param := range subParts[1:] {
			param = strings.TrimSpace(param)
			if strings.HasPrefix(param, "q=") {
				if qVal, err := strconv.ParseFloat(strings.TrimPrefix(param, "q="), 64); err == nil {
					quality = qVal
				}
			}
		}

		entries = append(entries, langEntry{tag: tag, quality: quality})
	}

	// Find best available match
	for _, e := range entries {
		// Exact match: "id-id" or "id"
		if bundle.HasLocale(e.tag) {
			return e.tag
		}
		// Base match: "id-id" -> "id"
		if idx := strings.IndexByte(e.tag, '-'); idx != -1 {
			baseTag := e.tag[:idx]
			if bundle.HasLocale(baseTag) {
				return baseTag
			}
		}
	}

	return ""
}
