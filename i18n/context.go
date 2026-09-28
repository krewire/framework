package i18n

import "context"

type contextKey struct{}

var localeCtxKey = contextKey{}

// WithLocale returns a copy of parent context in which the locale is set.
func WithLocale(ctx context.Context, locale string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, localeCtxKey, locale)
}

// Locale returns the locale stored in ctx, or "" if not set.
func Locale(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(localeCtxKey).(string); ok {
		return v
	}
	return ""
}

// LocaleFromContext is an alias for Locale.
func LocaleFromContext(ctx context.Context) string {
	return Locale(ctx)
}
