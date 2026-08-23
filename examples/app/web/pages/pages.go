// Package pages defines the application's SSR page definitions
// (FRK-STR-006): data + mounts registered at assembly time.
package pages

import (
	"net/http"
	"time"

	"github.com/krewire/framework/web"
)

// Register wires all application pages into the web app.
func Register(a *web.App) {
	a.Page(web.PageSpec{
		Path:   "/",
		Title:  "Krewire Monolith",
		Layout: "Shell",
		Root:   "Hero",
		Data: map[string]any{
			"badge":    "v0.6.0",
			"title":    "One model, every mode",
			"subtitle": "SSR, API, and static export from a single runtime.",
			"cta_left": map[string]any{"label": "Read the docs", "href": "/docs"},
		},
		EmitProps: true,
		Scripts:   []string{"/static/app.js"},
	})

	a.Page(web.PageSpec{
		Path:   "/greet",
		Title:  "Greet",
		Layout: "Shell",
		Root:   "Hero",
		DataFunc: func(r *http.Request) (any, error) {
			name := r.URL.Query().Get("name")
			if name == "" {
				name = "world"
			}
			return map[string]any{
				"badge":    time.Now().Format("15:04:05"),
				"title":    "Hello, " + name,
				"subtitle": "This page is rendered per request.",
			}, nil
		},
	})
}
