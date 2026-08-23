package web

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/krewire/framework/web/ssg"
)

// Handler returns the assembled http.Handler: routes, middleware, pages, and
// static mounts. It is safe for tests and embedding in larger servers.
func (a *App) Handler() http.Handler {
	site := a.site()
	for _, p := range a.pages {
		spec := p
		a.router.Get(spec.Path, func(w http.ResponseWriter, req *http.Request, _ Params) {
			data, err := pageData(spec, req)
			if err != nil {
				Error(w, err)
				return
			}
			body, err := site.RenderPage(&ssg.Page{
				Path:    spec.Path,
				Title:   spec.Title,
				Layout:  spec.Layout,
				Root:    spec.Root,
				Data:    data,
				Props:   propsFor(spec, data),
				Scripts: spec.Scripts,
			})
			if err != nil {
				Error(w, err)
				return
			}
			HTML(w, http.StatusOK, body)
		})
	}
	for _, name := range site.Assets() {
		assetName := name
		assetBody, _ := site.AssetBody(name)
		a.router.Get("/"+assetName, func(w http.ResponseWriter, _ *http.Request, _ Params) {
			serveAsset(w, assetName, assetBody)
		})
	}
	for _, st := range a.statics {
		a.router.StaticFS(st.prefix, st.fsys)
	}
	return a.router
}

// serveAsset writes an embedded asset body with a content type derived from
// its extension.
func serveAsset(w http.ResponseWriter, name, body string) {
	switch filepath.Ext(name) {
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case ".js":
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	_, _ = io.WriteString(w, body)
}

// Run serves the App over HTTP on addr, shutting down gracefully on
// SIGINT/SIGTERM.
func (a *App) Run(addr string) error {
	srv := &http.Server{Addr: addr, Handler: a.Handler()}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("krewire server started", "addr", addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close()
		return err
	}
	slog.Info("krewire server stopped")
	return nil
}
