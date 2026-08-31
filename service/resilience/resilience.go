// Package resilience provides circuit breaker, retry, timeout, and bulkhead
// primitives (KWF-L5H2F FRK-SVC-030/031/032/033). All are opt-in and compose
// with context for cancellation.
package resilience

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"
)

// CircuitBreaker is a state machine: closed → open → half-open (FRK-SVC-030).
type CircuitBreaker struct {
	mu            sync.Mutex
	name          string
	state         cbState
	failures      int
	threshold     int
	timeout       time.Duration
	halfOpenMax   int
	successes     int
	lastFailure   time.Time
	onStateChange func(name string, from, to cbState)
}

type cbState int

const (
	stateClosed cbState = iota
	stateOpen
	stateHalfOpen
)

// CircuitBreakerConfig configures a breaker (FRK-SVC-030).
type CircuitBreakerConfig struct {
	Name          string
	Threshold     int           // failures before opening
	Timeout       time.Duration // time in open before half-open
	HalfOpenMax   int           // successes in half-open before closing
	OnStateChange func(name string, from, to cbState)
}

// NewCircuitBreaker creates a closed breaker (FRK-SVC-030).
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.Threshold <= 0 {
		cfg.Threshold = 5
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.HalfOpenMax <= 0 {
		cfg.HalfOpenMax = 1
	}
	return &CircuitBreaker{
		name:          cfg.Name,
		threshold:     cfg.Threshold,
		timeout:       cfg.Timeout,
		halfOpenMax:   cfg.HalfOpenMax,
		onStateChange: cfg.OnStateChange,
	}
}

// ErrCircuitOpen signals the breaker is open (FRK-SVC-030).
type ErrCircuitOpen struct{ Name string }

func (e ErrCircuitOpen) Error() string { return "circuit breaker " + e.Name + " is open" }

// Do runs fn through the breaker (FRK-SVC-030).
func (cb *CircuitBreaker) Do(ctx context.Context, fn func() error) error {
	if !cb.allow() {
		return ErrCircuitOpen{Name: cb.name}
	}
	err := fn()
	cb.record(err == nil)
	return err
}

func (cb *CircuitBreaker) allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case stateClosed:
		return true
	case stateOpen:
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.transition(stateHalfOpen)
			cb.successes = 0
			return true
		}
		return false
	case stateHalfOpen:
		return true
	}
	return false
}

func (cb *CircuitBreaker) record(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if success {
		switch cb.state {
		case stateHalfOpen:
			cb.successes++
			if cb.successes >= cb.halfOpenMax {
				cb.transition(stateClosed)
				cb.failures = 0
			}
		case stateClosed:
			cb.failures = 0
		}
		return
	}
	switch cb.state {
	case stateClosed:
		cb.failures++
		if cb.failures >= cb.threshold {
			cb.transition(stateOpen)
			cb.lastFailure = time.Now()
		}
	case stateHalfOpen:
		cb.transition(stateOpen)
		cb.lastFailure = time.Now()
	}
}

func (cb *CircuitBreaker) transition(to cbState) {
	from := cb.state
	cb.state = to
	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, from, to)
	}
}

// State returns the current state name for observability.
func (cb *CircuitBreaker) State() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case stateClosed:
		return "closed"
	case stateOpen:
		return "open"
	case stateHalfOpen:
		return "half-open"
	}
	return "unknown"
}

// RetryPolicy configures retry behavior (FRK-SVC-031).
type RetryPolicy struct {
	MaxAttempts int
	Backoff     time.Duration
	MaxBackoff  time.Duration
	Jitter      float64
	RetryIf     func(error) bool
}

// DefaultRetryPolicy returns a sensible default (FRK-SVC-031).
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 3,
		Backoff:     100 * time.Millisecond,
		MaxBackoff:  10 * time.Second,
		Jitter:      0.2,
		RetryIf:     func(error) bool { return true },
	}
}

// Do retries fn according to the policy, respecting context (FRK-SVC-031).
func (p RetryPolicy) Do(ctx context.Context, fn func() error) error {
	var err error
	backoff := p.Backoff
	for attempt := 0; attempt < p.MaxAttempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if p.RetryIf != nil && !p.RetryIf(err) {
			return err
		}
		if attempt == p.MaxAttempts-1 {
			break
		}
		jitter := time.Duration(float64(backoff) * (1 + rand.Float64()*p.Jitter))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(jitter):
		}
		if backoff < p.MaxBackoff {
			backoff *= 2
		}
	}
	return err
}

// Bulkhead bounds concurrent calls (FRK-SVC-033).
type Bulkhead struct {
	sem    chan struct{}
	closed chan struct{}
}

// NewBulkhead creates a bulkhead with the given concurrency limit (FRK-SVC-033).
func NewBulkhead(limit int) *Bulkhead {
	return &Bulkhead{
		sem:    make(chan struct{}, limit),
		closed: make(chan struct{}),
	}
}

// ErrBulkheadFull signals the bulkhead is at capacity (FRK-SVC-033).
type ErrBulkheadFull struct{}

func (ErrBulkheadFull) Error() string { return "bulkhead full" }

// Do runs fn if capacity allows (FRK-SVC-033).
func (b *Bulkhead) Do(ctx context.Context, fn func() error) error {
	select {
	case b.sem <- struct{}{}:
		defer func() { <-b.sem }()
		return fn()
	case <-ctx.Done():
		return ctx.Err()
	case <-b.closed:
		return errors.New("bulkhead closed")
	}
}

// Close shuts down the bulkhead.
func (b *Bulkhead) Close() { close(b.closed) }
