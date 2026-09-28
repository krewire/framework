package web

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/krewire/framework/runner"
	"github.com/krewire/framework/service"
	"github.com/krewire/framework/web/ssg"
)

const DefaultShutdownTimeout = 10 * time.Second

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

// Serve runs the HTTP server listening on addr until ctx is cancelled, then
// shuts down gracefully with a 10-second timeout. Timeouts are configured to
// prevent connection exhaustion and Slowloris attacks.
func (a *App) Serve(ctx context.Context, addr string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      a.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("krewire server started", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), DefaultShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close()
		return err
	}
	slog.Info("krewire server stopped")
	return nil
}

// Runner adapts the App into a runner.Runner that serves HTTP on addr
// within an Application lifecycle.
func (a *App) Runner(addr string) runner.Runner {
	return runner.Func(func(ctx context.Context, _ service.Registry) error {
		return a.Serve(ctx, addr)
	})
}

// Run serves the App over HTTP on addr, shutting down gracefully on
// SIGINT/SIGTERM.
func (a *App) Run(addr string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return a.Serve(ctx, addr)
}
