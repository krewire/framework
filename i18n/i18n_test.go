package i18n_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/krewire/framework/app"
	"github.com/krewire/framework/i18n"
	svcconfig "github.com/krewire/framework/service/config"
)

func sampleFileSystem() fstest.MapFS {
	return fstest.MapFS{
		// Indonesian
		"lang/id/auth.json": &fstest.MapFile{
			Data: []byte(`{
				"login": "Masuk ke Akun",
				"btn": {
					"submit": "Kirim"
				}
			}`),
		},
		"lang/id.json": &fstest.MapFile{
			Data: []byte(`{
				"app_title": "Aplikasi Krewire",
				"auth": {
					"forgot": "Lupa Kata Sandi",
					"login": "Masuk Fallback"
				}
			}`),
		},

		// English (Default fallback)
		"lang/en/auth.json": &fstest.MapFile{
			Data: []byte(`{
				"login": "Log in",
				"logout": "Log out"
			}`),
		},
		"lang/en.json": &fstest.MapFile{
			Data: []byte(`{
				"app_title": "Krewire App",
				"global_msg": "Hello {name}",
				"messages": {
					"zero": "No new messages",
					"one": "1 new message",
					"other": "{count} new messages"
				},
				"items": "No items | One item | {count} items"
			}`),
		},

		// French (Single file only)
		"lang/fr.json": &fstest.MapFile{
			Data: []byte(`{
				"welcome": "Bienvenue {{name}}!"
			}`),
		},

		// Direct file match
		"lang/id/welcome.json": &fstest.MapFile{
			Data: []byte(`{
				"message": "Selamat datang :name!"
			}`),
		},
	}
}

func TestBundle_NamespacedKeyWithFallback(t *testing.T) {
	fsys := sampleFileSystem()
	bundle, err := i18n.New(fsys,
		i18n.WithBasePath("lang"),
		i18n.WithDefaultLocale("en"),
		i18n.WithFallbackLocale("en"),
	)
	if err != nil {
		t.Fatalf("unexpected error creating bundle: %v", err)
	}

	if bundle.DefaultLocale() != "en" {
		t.Errorf("expected default locale 'en', got '%s'", bundle.DefaultLocale())
	}
	if bundle.FallbackLocale() != "en" {
		t.Errorf("expected fallback locale 'en', got '%s'", bundle.FallbackLocale())
	}

	locales := bundle.Locales()
	if len(locales) < 3 {
		t.Errorf("expected at least 3 locales, got %v", locales)
	}
	if !bundle.HasLocale("id") || !bundle.HasLocale("en") || !bundle.HasLocale("fr") {
		t.Errorf("expected locales to contain id, en, fr; got %v", locales)
	}

	// 1. Target key exists in namespaced file lang/id/auth.json (takes priority over lang/id.json)
	got := bundle.Translate("id", "auth.login")
	if got != "Masuk ke Akun" {
		t.Errorf("expected 'Masuk ke Akun', got '%s'", got)
	}

	// 2. Nested subkey in namespaced file
	got = bundle.Translate("id", "auth.btn.submit")
	if got != "Kirim" {
		t.Errorf("expected 'Kirim', got '%s'", got)
	}

	// 3. Key missing in namespaced file, but present in fallback file lang/id.json
	got = bundle.Translate("id", "auth.forgot")
	if got != "Lupa Kata Sandi" {
		t.Errorf("expected 'Lupa Kata Sandi', got '%s'", got)
	}

	// 4. Root key in fallback file lang/id.json
	got = bundle.Translate("id", "app_title")
	if got != "Aplikasi Krewire" {
		t.Errorf("expected 'Aplikasi Krewire', got '%s'", got)
	}

	// 5. Key missing in target locale 'id', falls back to default locale 'en' (namespaced file)
	got = bundle.Translate("id", "auth.logout")
	if got != "Log out" {
		t.Errorf("expected 'Log out', got '%s'", got)
	}

	// 6. Key missing in target locale 'id', falls back to default locale 'en' (fallback file)
	got = bundle.Translate("id", "global_msg", "name", "Budi")
	if got != "Hello Budi" {
		t.Errorf("expected 'Hello Budi', got '%s'", got)
	}

	// 7. Non-existent key returns key name
	got = bundle.Translate("id", "non.existent.key")
	if got != "non.existent.key" {
		t.Errorf("expected 'non.existent.key', got '%s'", got)
	}
}

func TestBundle_SingleFileAndDirectLookup(t *testing.T) {
	fsys := sampleFileSystem()
	bundle, err := i18n.New(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test single file fr.json with {{name}} template
	got := bundle.Translate("fr", "welcome", map[string]any{"name": "Pierre"})
	if got != "Bienvenue Pierre!" {
		t.Errorf("expected 'Bienvenue Pierre!', got '%s'", got)
	}

	// Test direct file welcome.json with :name template
	got = bundle.Translate("id", "welcome", "name", "Budi")
	if got != "Selamat datang Budi!" {
		t.Errorf("expected 'Selamat datang Budi!', got '%s'", got)
	}
}

func TestBundle_AddTranslation(t *testing.T) {
	bundle, err := i18n.New(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bundle.AddTranslation("es", "auth", map[string]any{
		"login": "Iniciar sesión",
	})
	bundle.AddTranslation("es", "", map[string]any{
		"title": "Bienvenido",
	})

	if !bundle.HasLocale("es") {
		t.Error("expected locale es to be registered")
	}
	if got := bundle.Translate("es", "auth.login"); got != "Iniciar sesión" {
		t.Errorf("expected 'Iniciar sesión', got '%s'", got)
	}
	if got := bundle.Translate("es", "title"); got != "Bienvenido" {
		t.Errorf("expected 'Bienvenido', got '%s'", got)
	}
}

func TestBundle_Pluralization(t *testing.T) {
	fsys := sampleFileSystem()
	bundle, err := i18n.New(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Map-based pluralization
	if got := bundle.Translate("en", "messages", "count", 0); got != "No new messages" {
		t.Errorf("count=0: expected 'No new messages', got '%s'", got)
	}
	if got := bundle.Translate("en", "messages", "count", int64(1)); got != "1 new message" {
		t.Errorf("count=1: expected '1 new message', got '%s'", got)
	}
	if got := bundle.Translate("en", "messages", "count", float64(5)); got != "5 new messages" {
		t.Errorf("count=5: expected '5 new messages', got '%s'", got)
	}
	if got := bundle.Translate("en", "messages", "count", "10"); got != "10 new messages" {
		t.Errorf("count='10': expected '10 new messages', got '%s'", got)
	}

	// Pipe-separated pluralization
	if got := bundle.Translate("en", "items", "count", 0); got != "No items" {
		t.Errorf("count=0: expected 'No items', got '%s'", got)
	}
	if got := bundle.Translate("en", "items", "count", 1); got != "One item" {
		t.Errorf("count=1: expected 'One item', got '%s'", got)
	}
	if got := bundle.Translate("en", "items", "count", 3); got != "3 items" {
		t.Errorf("count=3: expected '3 items', got '%s'", got)
	}
}

func TestBundle_ContextAndTranslator(t *testing.T) {
	fsys := sampleFileSystem()
	bundle, err := i18n.New(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Context helpers
	if i18n.Locale(nil) != "" {
		t.Errorf("expected empty string for nil context, got '%s'", i18n.Locale(nil))
	}
	ctx := i18n.WithLocale(nil, "id")
	if i18n.LocaleFromContext(ctx) != "id" {
		t.Errorf("expected 'id', got '%s'", i18n.LocaleFromContext(ctx))
	}
	if got := bundle.T(ctx, "auth.login"); got != "Masuk ke Akun" {
		t.Errorf("expected 'Masuk ke Akun', got '%s'", got)
	}

	// Translator instance
	tr := bundle.ForLocale("id")
	if tr.Locale() != "id" {
		t.Errorf("expected 'id', got '%s'", tr.Locale())
	}
	if tr.Bundle() != bundle {
		t.Errorf("expected tr.Bundle() to match original bundle")
	}
	if got := tr.Translate("auth.login"); got != "Masuk ke Akun" {
		t.Errorf("expected 'Masuk ke Akun', got '%s'", got)
	}
	if !tr.Has("auth.login") {
		t.Errorf("expected Has('auth.login') to be true")
	}

	tr.SetLocale("en")
	if got := tr.T("auth.login"); got != "Log in" {
		t.Errorf("expected 'Log in', got '%s'", got)
	}
}

func TestMiddleware(t *testing.T) {
	fsys := sampleFileSystem()
	bundle, err := i18n.New(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loc := i18n.Locale(r.Context())
		w.Write([]byte(loc))
	})

	mw := i18n.Middleware(bundle,
		i18n.WithQueryParam("locale"),
		i18n.WithCookieName("user_lang"),
		i18n.WithHeaderName("Accept-Language"),
	)
	server := mw(handler)

	// 1. Query parameter
	req1 := httptest.NewRequest("GET", "/?locale=id", nil)
	rr1 := httptest.NewRecorder()
	server.ServeHTTP(rr1, req1)
	if rr1.Body.String() != "id" {
		t.Errorf("query param: expected 'id', got '%s'", rr1.Body.String())
	}

	// 2. Cookie
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.AddCookie(&http.Cookie{Name: "user_lang", Value: "fr"})
	rr2 := httptest.NewRecorder()
	server.ServeHTTP(rr2, req2)
	if rr2.Body.String() != "fr" {
		t.Errorf("cookie: expected 'fr', got '%s'", rr2.Body.String())
	}

	// 3. Accept-Language header
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.Header.Set("Accept-Language", "id-ID,id;q=0.9,en-US;q=0.8")
	rr3 := httptest.NewRecorder()
	server.ServeHTTP(rr3, req3)
	if rr3.Body.String() != "id" {
		t.Errorf("header: expected 'id', got '%s'", rr3.Body.String())
	}

	// 4. Default fallback
	req4 := httptest.NewRequest("GET", "/", nil)
	rr4 := httptest.NewRecorder()
	server.ServeHTTP(rr4, req4)
	if rr4.Body.String() != "en" {
		t.Errorf("default: expected 'en', got '%s'", rr4.Body.String())
	}
}

func TestProvider(t *testing.T) {
	fsys := sampleFileSystem()
	bundle, err := i18n.New(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	container, err := app.NewApp(i18n.Provider(bundle)).Build()
	if err != nil {
		t.Fatalf("container build error: %v", err)
	}

	resolvedBundle, err := app.Resolve[*i18n.Bundle](container)
	if err != nil {
		t.Fatalf("failed to resolve *i18n.Bundle: %v", err)
	}
	if resolvedBundle == nil {
		t.Fatal("resolved bundle is nil")
	}

	resolvedTranslator, err := app.Resolve[*i18n.Translator](container)
	if err != nil {
		t.Fatalf("failed to resolve *i18n.Translator: %v", err)
	}
	if resolvedTranslator == nil {
		t.Fatal("resolved translator is nil")
	}
	if resolvedTranslator.T("auth.login") != "Log in" {
		t.Errorf("expected 'Log in', got '%s'", resolvedTranslator.T("auth.login"))
	}
}

func TestBundle_Concurrency(t *testing.T) {
	fsys := sampleFileSystem()
	bundle, err := i18n.New(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var wg sync.WaitGroup
	ctx := i18n.WithLocale(context.Background(), "id")

	for i := 0; i < 50; i++ {
		wg.Add(3)

		go func() {
			defer wg.Done()
			_ = bundle.Translate("id", "auth.login")
			_ = bundle.Has("id", "auth.login")
		}()

		go func() {
			defer wg.Done()
			_ = bundle.T(ctx, "auth.login")
			_ = bundle.Translate("en", "messages", "count", 2)
		}()

		go func() {
			defer wg.Done()
			tr := bundle.ForLocale("fr")
			_ = tr.T("welcome", "name", "User")
		}()
	}

	wg.Wait()
}

func TestConfigAndCenter(t *testing.T) {
	fsys := sampleFileSystem()
	cfg := i18n.DefaultConfig()
	cfg.DefaultLocale = "id"
	cfg.FallbackLocale = "en"
	cfg.Locales = []string{"id", "en", "es"}

	bundle, err := i18n.NewFromConfig(fsys, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if bundle.DefaultLocale() != "id" {
		t.Errorf("expected default locale 'id', got '%s'", bundle.DefaultLocale())
	}
	if !bundle.HasLocale("es") {
		t.Error("expected locale 'es' to be registered from config")
	}

	// Test framework config center integration
	center := svcconfig.NewMemoryCenter()
	ctx := context.Background()

	// Store YAML config in center
	yamlData := `
default_locale: fr
fallback_locale: en
locales: [fr, en, de]
`
	if err := center.Set(ctx, "i18n.settings", yamlData); err != nil {
		t.Fatalf("failed to set center config: %v", err)
	}

	loadedCfg, err := i18n.LoadConfigFromCenter(ctx, center, "i18n.settings")
	if err != nil {
		t.Fatalf("failed to load config from center: %v", err)
	}
	if loadedCfg.DefaultLocale != "fr" {
		t.Errorf("expected loaded default locale 'fr', got '%s'", loadedCfg.DefaultLocale)
	}

	// Watch center for hot-reload
	cancel, err := i18n.WatchCenter(ctx, center, "i18n.settings", bundle)
	if err != nil {
		t.Fatalf("failed to watch center: %v", err)
	}
	defer cancel()

	// Update in center
	updatedData := `{"default_locale": "de", "locales": ["de", "en"]}`
	if err := center.Set(ctx, "i18n.settings", updatedData); err != nil {
		t.Fatalf("failed to update center: %v", err)
	}

	// Give watcher a moment to process change
	for i := 0; i < 20; i++ {
		if bundle.DefaultLocale() == "de" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if bundle.DefaultLocale() != "de" {
		t.Errorf("expected hot-reloaded locale 'de', got '%s'", bundle.DefaultLocale())
	}
}

func TestProviderWithConfig(t *testing.T) {
	fsys := sampleFileSystem()
	cfg := i18n.Config{
		DefaultLocale:  "id",
		FallbackLocale: "en",
	}

	bundle, err := i18n.New(fsys)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	container, err := app.NewApp(i18n.ProviderWithConfig(bundle, cfg)).Build()
	if err != nil {
		t.Fatalf("failed to build container: %v", err)
	}

	resolvedTranslator, err := app.Resolve[*i18n.Translator](container)
	if err != nil {
		t.Fatalf("failed to resolve translator: %v", err)
	}
	if resolvedTranslator.Locale() != "id" {
		t.Errorf("expected locale 'id', got '%s'", resolvedTranslator.Locale())
	}
}
