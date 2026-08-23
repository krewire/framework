package app

import (
	"fmt"
	"reflect"
	"time"
)

// Provider declares the two assembly phases an application module must
// implement (FRK-CNT-020): Register binds every dependency into the container
// before any Boot runs.
type Provider interface {
	Register(c *Container) error
	Boot(c *Container) error
}

// Registerer is a convenience interface for phases that need no boot work;
// ProviderFunc adapts it.
type Registerer interface {
	Register(c *Container) error
}

// ProviderFunc adapts a Register-only function into a Provider.
func ProviderFunc(fn func(c *Container) error) Provider {
	return providerFunc(fn)
}

type providerFunc func(c *Container) error

func (p providerFunc) Register(c *Container) error { return p(c) }
func (p providerFunc) Boot(c *Container) error     { return nil }

// App aggregates providers: every Register runs in order, then every Boot runs
// in order, then the container is ready for resolution. Create one with New.
type App struct {
	providers []Provider
	container *Container
	opts      []Option
}

// NewApp returns an App assembling the given providers. No providers is valid
// and yields an empty container. Register and Boot only run on Build or Test
// (FRK-CNT-012).
func NewApp(providers ...Provider) *App {
	return &App{
		providers: providers,
		container: New(),
		opts:      nil,
	}
}

// Options applies container options to the assembly.
func (a *App) Options(opts ...Option) *App {
	a.opts = append(a.opts, opts...)
	return a
}

// Build runs Register for every provider in order, then Boot for every
// provider in order, and returns the assembled container (FRK-CNT-013). It
// emits a boot summary record (FRK-CNT-031) once assembly finishes.
func (a *App) Build() (*Container, error) {
	start := time.Now()
	c := New(a.opts...)
	if err := a.runRegister(c); err != nil {
		return nil, err
	}
	if err := a.runBoot(c); err != nil {
		return nil, err
	}
	if c.logger != nil {
		c.logger.Info(
			"boot",
			"providers", len(a.providers),
			"bindings", len(c.Bindings()),
			"duration", time.Since(start),
		)
	}
	return c, nil
}

// Test assembles the container the same way as Build (FRK-CNT-040). Overrides
// are arbitrary factories `func(deps...) (T, error)` applied after Register and
// before Boot, replacing the corresponding bindings. Boot still runs, letting
// all providers observe the overridden container.
func (a *App) Test(overrides ...any) (*Container, error) {
	c := New(a.opts...)
	if err := a.runRegister(c); err != nil {
		return nil, err
	}
	for _, factory := range overrides {
		if err := applyOverride(c, factory); err != nil {
			return nil, fmt.Errorf("app: test override: %w", err)
		}
	}
	if err := a.runBoot(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (a *App) runRegister(c *Container) error {
	for _, p := range a.providers {
		if err := p.Register(c); err != nil {
			return fmt.Errorf("app: provider %s: register: %w", providerName(p), err)
		}
	}
	return nil
}

func (a *App) runBoot(c *Container) error {
	for _, p := range a.providers {
		if err := p.Boot(c); err != nil {
			return fmt.Errorf("app: provider %s: boot: %w", providerName(p), err)
		}
	}
	return nil
}

func providerName(p Provider) string {
	return fmt.Sprintf("%T", p)
}

// applyOverride applies a single test override factory, discovering the target
// type from the factory's return type.
func applyOverride(c *Container, factory any) error {
	fv := reflect.ValueOf(factory)
	if !fv.IsValid() || fv.Kind() != reflect.Func {
		return fmt.Errorf("override must be a func(deps...) (T, error)")
	}
	ft := fv.Type()
	if ft.NumOut() == 0 || ft.NumOut() > 2 {
		return fmt.Errorf("override return type must be (T, error) or T")
	}
	typ := ft.Out(0)
	if err := c.register(factory, typ, false); err != nil {
		return err
	}
	return nil
}
