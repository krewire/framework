package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/krewire/framework/runner"
	"github.com/krewire/framework/service"
)

const DefaultStopTimeout = 10 * time.Second

// Runner is the runtime contract supplied to Application.Run.
type Runner = runner.Runner

// RunnerFunc adapts a function into a Runner.
type RunnerFunc = runner.Func

// Application manages provider registration, lifecycle, and runtime runners.
type Application struct {
	container        *Container
	app              *App
	appStarted       bool
	serviceProviders []service.Provider
	startedProviders []service.Provider
	serviceMu        sync.Mutex
}

// NewApplication creates an application with an empty service container.
func NewApplication() *Application {
	return &Application{container: New()}
}

// Container returns the underlying DI container.
func (a *Application) Container() *Container { return a.container }

// App returns the provider aggregator that built this Application, if any.
func (a *Application) App() *App { return a.app }

// Use adds service providers in startup order. Providers must have unique names.
func (a *Application) Use(providers ...service.Provider) error {
	if a == nil {
		return fmt.Errorf("app: nil application")
	}
	a.serviceMu.Lock()
	defer a.serviceMu.Unlock()
	seen := make(map[string]struct{}, len(a.serviceProviders))
	for _, existing := range a.serviceProviders {
		seen[existing.Name()] = struct{}{}
	}
	for _, provider := range providers {
		if provider == nil {
			return fmt.Errorf("app: provider must not be nil")
		}
		name := provider.Name()
		if name == "" {
			return fmt.Errorf("app: provider name is required")
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("app: provider %q already registered", name)
		}
		seen[name] = struct{}{}
		a.serviceProviders = append(a.serviceProviders, provider)
	}
	return nil
}

// Bootstrap builds the container from providers (Register then Boot) and starts
// provider Starters in registration order. Any failure runs Stop and returns.
func (a *Application) Bootstrap(ctx context.Context) error {
	if a == nil {
		return fmt.Errorf("app: nil application")
	}
	if a.container == nil {
		a.container = New()
	}
	if a.app != nil && !a.appStarted {
		if err := a.app.Start(ctx, a.container); err != nil {
			return err
		}
		a.appStarted = true
	}
	a.serviceMu.Lock()
	defer a.serviceMu.Unlock()
	for _, provider := range a.serviceProviders {
		if err := provider.Register(a.container); err != nil {
			return fmt.Errorf("app: register %q: %w", provider.Name(), err)
		}
	}
	for _, provider := range a.serviceProviders {
		if starter, ok := provider.(service.Starter); ok {
			if err := starter.Start(ctx, a.container); err != nil {
				return fmt.Errorf("app: start %q: %w", provider.Name(), err)
			}
			a.startedProviders = append(a.startedProviders, provider)
		}
	}
	return nil
}

// Stop runs the shutdown lifecycle: hooks OnStop reverse then provider
// Stoppers reverse, with a 10s timeout derived from parent if needed.
func (a *Application) Stop(ctx context.Context) error {
	if a == nil {
		return fmt.Errorf("app: nil application")
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultStopTimeout)
		defer cancel()
	}
	serviceErr := a.stopStartedProviders(ctx)
	var appErr error
	if a.app != nil && a.container != nil {
		appErr = a.app.Stop(ctx, a.container)
		a.appStarted = false
	}
	return errors.Join(serviceErr, appErr)
}

// Shutdown stops started providers in reverse startup order.
func (a *Application) Shutdown(ctx context.Context) error {
	return a.Stop(ctx)
}

func (a *Application) stopStartedProviders(ctx context.Context) error {
	a.serviceMu.Lock()
	started := append([]service.Provider(nil), a.startedProviders...)
	a.startedProviders = nil
	a.serviceMu.Unlock()
	var stopErr error
	for i := len(started) - 1; i >= 0; i-- {
		if stopper, ok := started[i].(service.Stopper); ok {
			if err := stopper.Stop(ctx, a.container); err != nil {
				stopErr = errors.Join(stopErr, fmt.Errorf("app: stop %q: %w", started[i].Name(), err))
			}
		}
	}
	return stopErr
}

// Run executes optional runners after bootstrapping, or blocks until ctx is
// cancelled or a SIGINT/SIGTERM is received, then gracefully stops all providers.
func (a *Application) Run(ctx context.Context, runners ...Runner) error {
	if a == nil {
		return fmt.Errorf("app: nil application")
	}
	if err := a.Bootstrap(ctx); err != nil {
		return err
	}
	if len(runners) > 0 {
		var (
			wg      sync.WaitGroup
			mu      sync.Mutex
			runErrs []error
		)
		runCtx, cancelRunners := context.WithCancel(ctx)
		defer cancelRunners()

		for _, r := range runners {
			if r == nil {
				continue
			}
			workload := r
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := workload.Run(runCtx, a.container); err != nil && !errors.Is(err, context.Canceled) {
					mu.Lock()
					runErrs = append(runErrs, err)
					mu.Unlock()
					cancelRunners()
				}
			}()
		}
		wg.Wait()
		stopErr := a.Stop(ctx)
		if stopErr != nil {
			runErrs = append(runErrs, stopErr)
		}
		return errors.Join(runErrs...)
	}
	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()
	return a.Stop(ctx)
}
