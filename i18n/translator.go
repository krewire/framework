package i18n

import "sync"

// Translator provides localized translation methods bound to a specific locale.
type Translator struct {
	mu     sync.RWMutex
	bundle *Bundle
	locale string
}

// NewTranslator creates a Translator bound to bundle and locale.
func NewTranslator(bundle *Bundle, locale string) *Translator {
	if locale == "" && bundle != nil {
		locale = bundle.DefaultLocale()
	}
	return &Translator{
		bundle: bundle,
		locale: locale,
	}
}

// ForLocale creates a Translator bound to the given locale.
func (b *Bundle) ForLocale(locale string) *Translator {
	return NewTranslator(b, locale)
}

// Locale returns the currently configured locale for this translator.
func (t *Translator) Locale() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.locale
}

// SetLocale changes the target locale of this translator.
func (t *Translator) SetLocale(locale string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.locale = locale
}

// T translates key using the translator's bound locale.
func (t *Translator) T(key string, args ...any) string {
	t.mu.RLock()
	loc := t.locale
	bundle := t.bundle
	t.mu.RUnlock()

	if bundle == nil {
		return key
	}
	return bundle.Translate(loc, key, args...)
}

// Translate is an alias for T.
func (t *Translator) Translate(key string, args ...any) string {
	return t.T(key, args...)
}

// Has checks if key exists in the translator's bound locale.
func (t *Translator) Has(key string) bool {
	t.mu.RLock()
	loc := t.locale
	bundle := t.bundle
	t.mu.RUnlock()

	if bundle == nil {
		return false
	}
	return bundle.Has(loc, key)
}

// Bundle returns the underlying Bundle.
func (t *Translator) Bundle() *Bundle {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.bundle
}
