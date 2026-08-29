package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Application is the built container with lifecycle management. It is the
// canonical entrypoint for an app process: Bootstrap builds, starts the
// lifecycle, then Run blocks until graceful shutdown.
type Application struct {
	container *Container
	app       *App
}

// Bootstrap builds the container from providers (Register then Boot) and starts
// the lifecycle (provider Starters then App hooks). It returns the Application
// ready to serve. The caller must call Stop to shutdown gracefully.
func Bootstrap(ctx context.Context, providers ...Provider) (*Application, error) {
	return BootstrapWithOptions(ctx, nil, providers...)
}

// BootstrapWithOptions is like Bootstrap but applies container Options (e.g.
// WithLogger, WithTrace) before Build.
func BootstrapWithOptions(ctx context.Context, opts []Option, providers ...Provider) (*Application, error) {
	a := NewApp(providers...)
	if len(opts) > 0 {
		a.Options(opts...)
	}
	c, err := a.Build()
	if err != nil {
		return nil, err
	}
	if err := a.Start(ctx, c); err != nil {
		return nil, err
	}
	return &Application{container: c, app: a}, nil
}

// Container returns the underlying DI container.
func (a *Application) Container() *Container { return a.container }

// App returns the provider aggregator that built this Application.
func (a *Application) App() *App { return a.app }

// Stop runs the shutdown lifecycle: hooks OnStop reverse then provider
// Stoppers reverse, with a 10s timeout derived from parent if needed.
func (a *Application) Stop(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}
	return a.app.Stop(ctx, a.container)
}

// Run blocks until ctx is cancelled or a SIGINT/SIGTERM is received, then
// runs Stop. It is the canonical main.go entrypoint:
//
//	func main() {
//	    ctx := context.Background()
//	    app, err := app.Bootstrap(ctx, providers...)
//	    if err != nil { log.Fatal(err) }
//	    if err := app.Run(ctx); err != nil { log.Fatal(err) }
//	}
func (a *Application) Run(ctx context.Context) error {
	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()
	return a.Stop(context.Background())
}

// Run is a convenience helper that bootstraps, runs, and stops an application
// with signal-aware graceful shutdown. It is the one-liner entrypoint:
//
//	if err := app.Run(context.Background(), providers...); err != nil { ... }
func Run(ctx context.Context, providers ...Provider) error {
	return RunWithOptions(ctx, nil, providers...)
}

// RunWithOptions is like Run but applies container Options.
func RunWithOptions(ctx context.Context, opts []Option, providers ...Provider) error {
	a, err := BootstrapWithOptions(ctx, opts, providers...)
	if err != nil {
		return err
	}
	if err := a.Run(ctx); err != nil {
		return err
	}
	return nil
}
