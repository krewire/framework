package app

import (
	"context"
	"fmt"
	"time"
)

// Hook is a lifecycle hook. Either callback may be nil.
type Hook struct {
	OnStart func(context.Context) error
	OnStop  func(context.Context) error
}

// Starter is optionally implemented by a Provider to start long-running work
// after Boot. It is called during App.Start / Application.Start in registration order.
type Starter interface {
	Start(context.Context) error
}

// Stopper is optionally implemented by a Provider to stop gracefully during
// App.Stop / Application.Stop in reverse order.
type Stopper interface {
	Stop(context.Context) error
}

// AddHook appends a lifecycle hook. Hooks run after provider Starters and
// before provider Stoppers, preserving deterministic order.
func (a *App) AddHook(h Hook) *App {
	a.hooks = append(a.hooks, h)
	return a
}

// OnStart registers a start hook.
func (a *App) OnStart(fn func(context.Context) error) *App {
	return a.AddHook(Hook{OnStart: fn})
}

// OnStop registers a stop hook.
func (a *App) OnStop(fn func(context.Context) error) *App {
	return a.AddHook(Hook{OnStop: fn})
}

// Start runs lifecycle start callbacks: provider Starters in registration order,
// then App hooks OnStart in order. It requires a built container.
func (a *App) Start(ctx context.Context, c *Container) error {
	if c == nil {
		return fmt.Errorf("app: Start requires a built container")
	}
	// Providers that implement Starter.
	for _, p := range a.providers {
		if s, ok := p.(Starter); ok {
			name := providerName(p)
			start := time.Now()
			if err := s.Start(ctx); err != nil {
				return fmt.Errorf("app: provider %s: start: %w", name, err)
			}
			if c.logger != nil {
				c.logger.Info("start", "provider", name, "duration", time.Since(start))
			}
		}
	}
	// Hooks OnStart.
	for i, h := range a.hooks {
		if h.OnStart == nil {
			continue
		}
		start := time.Now()
		if err := h.OnStart(ctx); err != nil {
			return fmt.Errorf("app: hook %d: start: %w", i, err)
		}
		if c.logger != nil {
			c.logger.Info("start", "hook", i, "duration", time.Since(start))
		}
	}
	return nil
}

// Stop runs lifecycle stop callbacks in reverse order: App hooks OnStop reverse,
// then provider Stoppers reverse.
func (a *App) Stop(ctx context.Context, c *Container) error {
	if c == nil {
		return fmt.Errorf("app: Stop requires a built container")
	}
	var firstErr error
	// Hooks OnStop reverse.
	for i := len(a.hooks) - 1; i >= 0; i-- {
		h := a.hooks[i]
		if h.OnStop == nil {
			continue
		}
		start := time.Now()
		if err := h.OnStop(ctx); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("app: hook %d: stop: %w", i, err)
		}
		if c.logger != nil {
			c.logger.Info("stop", "hook", i, "duration", time.Since(start))
		}
	}
	// Providers Stop reverse.
	for i := len(a.providers) - 1; i >= 0; i-- {
		p := a.providers[i]
		if s, ok := p.(Stopper); ok {
			name := providerName(p)
			start := time.Now()
			if err := s.Stop(ctx); err != nil && firstErr == nil {
				firstErr = fmt.Errorf("app: provider %s: stop: %w", name, err)
			}
			if c.logger != nil {
				c.logger.Info("stop", "provider", name, "duration", time.Since(start))
			}
		}
	}
	return firstErr
}
