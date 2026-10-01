package i18n

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"
)

// Bundle stores loaded translation dictionaries and resolves translation keys
// with support for namespaced files lang/{locale}/{key}.json and fallback lang/{locale}.json.
type Bundle struct {
	mu             sync.RWMutex
	fsys           fs.FS
	basePath       string
	defaultLocale  string
	fallbackLocale string

	// files maps relative clean file path to unmarshaled JSON data.
	// e.g. "en/auth.json" -> map[string]any, "en.json" -> map[string]any.
	files map[string]any

	// locales lists all discovered locales.
	locales map[string]struct{}
}

// Option configures a Bundle.
type Option func(*Bundle)

// WithBasePath sets the relative root directory for translation files (default: "lang").
func WithBasePath(p string) Option {
	return func(b *Bundle) {
		b.basePath = strings.Trim(p, "/")
	}
}

// WithDefaultLocale sets the primary default locale (e.g. "en").
func WithDefaultLocale(locale string) Option {
	return func(b *Bundle) {
		if locale != "" {
			b.defaultLocale = strings.ToLower(locale)
		}
	}
}

// WithFallbackLocale sets the fallback locale when a key is not found in the target locale.
func WithFallbackLocale(locale string) Option {
	return func(b *Bundle) {
		if locale != "" {
			b.fallbackLocale = strings.ToLower(locale)
		}
	}
}

// New creates and initializes a new i18n Bundle from the provided filesystem.
// It preloads all translation JSON files found in the configured directory.
func New(fsys fs.FS, opts ...Option) (*Bundle, error) {
	b := &Bundle{
		fsys:           fsys,
		basePath:       "lang",
		defaultLocale:  "en",
		fallbackLocale: "en",
		files:          make(map[string]any),
		locales:        make(map[string]struct{}),
	}

	for _, opt := range opts {
		opt(b)
	}

	if b.fsys != nil {
		if err := b.Load(); err != nil {
			return nil, err
		}
	}

	return b, nil
}

// DefaultLocale returns the bundle's configured default locale.
func (b *Bundle) DefaultLocale() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.defaultLocale
}

// FallbackLocale returns the bundle's configured fallback locale.
func (b *Bundle) FallbackLocale() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.fallbackLocale
}

// Locales returns a sorted list of all available locales.
func (b *Bundle) Locales() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	res := make([]string, 0, len(b.locales))
	for loc := range b.locales {
		res = append(res, loc)
	}
	sort.Strings(res)
	return res
}

// HasLocale reports whether a given locale is present in the bundle.
func (b *Bundle) HasLocale(locale string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, ok := b.locales[strings.ToLower(locale)]
	return ok
}

// Load scans and parses all JSON files in the bundle filesystem.
func (b *Bundle) Load() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.files = make(map[string]any)
	if b.locales == nil {
		b.locales = make(map[string]struct{})
	}

	// Determine root directory to walk:
	// If basePath is specified and exists in fsys, walk from basePath.
	// Otherwise, if basePath is not found, try walking from root "."
	walkRoot := b.basePath
	if walkRoot != "" {
		if _, err := fs.Stat(b.fsys, walkRoot); err != nil {
			walkRoot = "."
		}
	} else {
		walkRoot = "."
	}

	err := fs.WalkDir(b.fsys, walkRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
			return nil
		}

		// Calculate relative path normalized to the base
		rel := p
		if walkRoot != "." {
			rel = strings.TrimPrefix(p, walkRoot)
			rel = strings.TrimPrefix(rel, "/")
		} else if b.basePath != "" && strings.HasPrefix(p, b.basePath+"/") {
			rel = strings.TrimPrefix(p, b.basePath+"/")
		}

		cleanRel := path.Clean(rel)

		// Read and unmarshal JSON
		data, err := fs.ReadFile(b.fsys, p)
		if err != nil {
			return fmt.Errorf("i18n: read %s: %w", p, err)
		}

		var parsed any
		if err := json.Unmarshal(data, &parsed); err != nil {
			return fmt.Errorf("i18n: parse %s: %w", p, err)
		}

		b.files[cleanRel] = parsed

		// Track locale name
		// For "id.json" -> "id"
		// For "id/auth.json" -> "id"
		parts := strings.Split(cleanRel, "/")
		if len(parts) > 1 {
			b.locales[strings.ToLower(parts[0])] = struct{}{}
		} else {
			baseName := strings.TrimSuffix(cleanRel, ".json")
			b.locales[strings.ToLower(baseName)] = struct{}{}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

// AddTranslation programmatically registers a translation under a specific locale and file/namespace.
func (b *Bundle) AddTranslation(locale, filename string, data map[string]any) {
	b.mu.Lock()
	defer b.mu.Unlock()

	loc := strings.ToLower(locale)
	b.locales[loc] = struct{}{}

	var cleanPath string
	if filename == "" || filename == loc || filename == loc+".json" {
		cleanPath = loc + ".json"
	} else {
		cleanName := strings.TrimSuffix(filename, ".json") + ".json"
		cleanPath = path.Join(loc, cleanName)
	}

	b.files[cleanPath] = data
}

// Translate resolves key for the specified locale and applies parameter interpolation.
// Lookup hierarchy:
//  1. Target locale:
//     a. lang/{locale}/{key}.json (namespaced file)
//     b. lang/{locale}.json (fallback file)
//  2. Fallback locale:
//     a. lang/{fallbackLocale}/{key}.json
//     b. lang/{fallbackLocale}.json
//  3. Default locale:
//     a. lang/{defaultLocale}/{key}.json
//     b. lang/{defaultLocale}.json
//  4. Returns key itself if not found.
func (b *Bundle) Translate(locale, key string, args ...any) string {
	params := normalizeParams(args)
	loc := strings.ToLower(locale)
	if loc == "" {
		loc = b.DefaultLocale()
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	raw, found := b.lookupLocked(loc, key)
	if !found {
		return key
	}

	resolved, ok := resolvePlural(raw, params)
	if !ok {
		return fmt.Sprintf("%v", raw)
	}

	return interpolate(resolved, params)
}

// Has checks whether a key exists in target locale or fallback locales.
func (b *Bundle) Has(locale, key string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	loc := strings.ToLower(locale)
	if loc == "" {
		loc = b.defaultLocale
	}

	_, found := b.lookupLocked(loc, key)
	return found
}

// T resolves a translation key taking the locale from context.Context.
func (b *Bundle) T(ctx context.Context, key string, args ...any) string {
	loc := Locale(ctx)
	if loc == "" {
		loc = b.DefaultLocale()
	}
	return b.Translate(loc, key, args...)
}

// lookupLocked resolves key following the locale fallback chain.
func (b *Bundle) lookupLocked(locale, key string) (any, bool) {
	// 1. Try target locale
	if val, ok := b.lookupInLocale(locale, key); ok {
		return val, true
	}

	// 2. Try fallback locale
	if b.fallbackLocale != "" && b.fallbackLocale != locale {
		if val, ok := b.lookupInLocale(b.fallbackLocale, key); ok {
			return val, true
		}
	}

	// 3. Try default locale
	if b.defaultLocale != "" && b.defaultLocale != locale && b.defaultLocale != b.fallbackLocale {
		if val, ok := b.lookupInLocale(b.defaultLocale, key); ok {
			return val, true
		}
	}

	return nil, false
}

// lookupInLocale resolves key within a single locale.
// Priority:
// 1. Namespaced file: lang/{locale}/{namespace}.json
// 2. Fallback single file: lang/{locale}.json
func (b *Bundle) lookupInLocale(locale, key string) (any, bool) {
	// Check for namespaced keys (e.g. "auth.login" or "nested.auth.login")
	parts := strings.Split(key, ".")

	// Strategy A: Namespaced file lookup: lang/{locale}/{key}.json
	// Try splitting at different dot depths, starting from largest namespace prefix
	// e.g. For "a.b.c", check:
	// 1. locale/a/b.json -> "c"
	// 2. locale/a.json -> "b.c"
	for i := len(parts) - 1; i >= 1; i-- {
		nsPath := strings.Join(parts[:i], "/") + ".json"
		subKey := strings.Join(parts[i:], ".")

		filePath := path.Join(locale, nsPath)
		if fileData, ok := b.files[filePath]; ok {
			if val, found := lookupInObject(fileData, subKey); found {
				return val, true
			}
		}
	}

	// If key has no dots (e.g. "welcome"), check if there is a file lang/{locale}/welcome.json
	if len(parts) == 1 {
		singleFilePath := path.Join(locale, key+".json")
		if fileData, ok := b.files[singleFilePath]; ok {
			// If file contains a primitive or root string
			if s, ok := fileData.(string); ok {
				return s, true
			}
			// If object, check if key is repeated or standard fields
			if val, found := lookupInObject(fileData, key); found {
				return val, true
			}
			if val, found := lookupInObject(fileData, "message"); found {
				return val, true
			}
			if val, found := lookupInObject(fileData, "value"); found {
				return val, true
			}
			return fileData, true
		}
	}

	// Strategy B: Fallback file: lang/{locale}.json
	fallbackFilePath := locale + ".json"
	if fileData, ok := b.files[fallbackFilePath]; ok {
		if val, found := lookupInObject(fileData, key); found {
			return val, true
		}
	}

	return nil, false
}

// lookupInObject traverses an unmarshaled JSON data structure by key or dot-path.
func lookupInObject(data any, keyPath string) (any, bool) {
	if data == nil || keyPath == "" {
		return nil, false
	}

	// 1. Exact match on current level
	if m, ok := data.(map[string]any); ok {
		if val, exists := m[keyPath]; exists {
			return val, true
		}
	}

	// 2. Dot-separated path traversal
	if strings.Contains(keyPath, ".") {
		parts := strings.Split(keyPath, ".")
		curr := data
		for _, part := range parts {
			m, ok := curr.(map[string]any)
			if !ok {
				return nil, false
			}
			next, exists := m[part]
			if !exists {
				return nil, false
			}
			curr = next
		}
		return curr, true
	}

	return nil, false
}
