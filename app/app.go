// Package app provides the Krewire app container: a typed dependency container
// and service-provider assembly (KWF-D1CNT). It is the single composition
// root of an application — every dependency is a typed binding, built and
// resolved at one explicit place, with deterministic order, observable
// construction, and test-time override seams.
//
// File responsibilities: app.go (container engine), provider.go (service
// provider assembly), errors.go (resolution error types), inspect.go
// (introspection), observe.go (tracer implementation).
package app

import (
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"time"
)

// Container is the composition root. Bindings are registered with Provide or
// Singleton and resolved with Resolve. It is safe for concurrent use after
// registration.
type Container struct {
	mu       sync.Mutex
	bindings map[reflect.Type]*binding
	order    []reflect.Type

	locked bool

	logger   *slog.Logger
	trace    bool
	tracer   Tracer
	resolved []reflect.Type
}

// binding holds one typed factory.
type binding struct {
	factory   any
	singleton bool
	built     bool
	value     reflect.Value
	err       error
}

// Option configures a Container.
type Option func(*Container)

// WithLogger sets the slog logger used for observability records.
func WithLogger(l *slog.Logger) Option {
	return func(c *Container) { c.logger = l }
}

// WithTrace enables a per-resolution trace record (FRK-CNT-030).
func WithTrace() Option {
	return func(c *Container) { c.trace = true }
}

// WithTracer installs a tracer receiving before/after resolution hooks
// (FRK-CNT-034).
func WithTracer(t Tracer) Option {
	return func(c *Container) { c.tracer = t }
}

// Tracer observes dependency resolution. Implementations must be safe for
// concurrent use.
type Tracer interface {
	BeforeResolve(typeName string)
	AfterResolve(typeName string, dur time.Duration, err error)
}

// New returns an empty Container with the given options applied.
func New(opts ...Option) *Container {
	c := &Container{
		bindings: make(map[reflect.Type]*binding),
		logger:   slog.Default(),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Provide registers a factory for T: an ordinary function
// `func(deps...) (T, error)` (or `func(deps...) T`) whose parameters are
// dependencies resolved from the container. Each Resolve builds a fresh value.
func Provide[T any](c *Container, factory any) error {
	return c.register(factory, typeOf[T](), false)
}

// Singleton registers a factory for T that builds lazily once and reuses the
// value (FRK-CNT-003).
func Singleton[T any](c *Container, factory any) error {
	return c.register(factory, typeOf[T](), true)
}

// Override replaces the binding for T before the first resolution (FRK-CNT-008).
func Override[T any](c *Container, factory any) error {
	return c.register(factory, typeOf[T](), false)
}

// Resolve returns the value bound to T, building it on demand. A missing
// binding, a failed dependency, or a constructor error returns an error
// carrying the full resolution path (FRK-CNT-004).
func Resolve[T any](c *Container) (T, error) {
	var zero T
	v, err := c.resolve(typeOf[T](), nil)
	if err != nil {
		return zero, err
	}
	return v.Interface().(T), nil
}

// typeOf returns the reflect.Type of T.
func typeOf[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

func (c *Container) register(factory any, typ reflect.Type, singleton bool) error {
	fv := reflect.ValueOf(factory)
	if fv.Kind() != reflect.Func {
		return fmt.Errorf("app: binding for %s is not a function", typ)
	}
	ft := fv.Type()
	switch ft.NumOut() {
	case 1:
	case 2:
		if !ft.Out(1).Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			return fmt.Errorf("app: binding for %s: second return must be error", typ)
		}
	default:
		return fmt.Errorf("app: binding for %s must return (T, error) or T", typ)
	}
	if !ft.Out(0).AssignableTo(typ) {
		return fmt.Errorf("app: binding for %s returns %s", typ, ft.Out(0))
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.locked {
		return fmt.Errorf("%w — cannot register %s", ErrLocked, typ)
	}
	if _, ok := c.bindings[typ]; ok && !singleton {
		// override replaces in place; Provide on an existing non-singleton
		// binding replaces too (documented override behavior).
	}
	c.bindings[typ] = &binding{factory: factory, singleton: singleton}
	c.order = append(c.order, typ)
	return nil
}

// resolve builds the value for typ, recording the resolution path for errors.
func (c *Container) resolve(typ reflect.Type, path []reflect.Type) (reflect.Value, error) {
	start := time.Now()
	typeName := typ.String()

	if c.tracer != nil {
		c.tracer.BeforeResolve(typeName)
	}

	c.mu.Lock()
	if c.locked {
		// already locked: recursion inside a factory is allowed.
	} else {
		c.locked = true
	}
	for _, t := range path {
		if t == typ {
			cycle := append(append([]reflect.Type(nil), path...), typ)
			names := make([]string, 0, len(cycle))
			for _, t := range cycle {
				names = append(names, t.String())
			}
			c.mu.Unlock()
			err := &CycleError{Path: names}
			c.after(typ, start, err)
			return reflect.Value{}, err
		}
	}
	b, ok := c.bindings[typ]
	if !ok {
		c.mu.Unlock()
		err := &ResolveError{
			Type: typeName,
			Path: pathNames(append(path, typ)),
			Err:  fmt.Errorf("no binding registered"),
		}
		c.after(typ, start, err)
		return reflect.Value{}, err
	}
	if b.singleton && b.built {
		c.mu.Unlock()
		c.after(typ, start, nil)
		return b.value, nil
	}
	c.mu.Unlock()

	v, err := c.call(b, typ, path)
	if err != nil {
		c.after(typ, start, err)
		return reflect.Value{}, err
	}

	c.mu.Lock()
	if b.singleton && !b.built {
		b.built = true
		b.value = v
	}
	c.resolved = append(c.resolved, typ)
	c.mu.Unlock()

	c.after(typ, start, nil)
	return v, nil
}

func (c *Container) after(typ reflect.Type, start time.Time, err error) {
	if c.tracer != nil {
		c.tracer.AfterResolve(typ.String(), time.Since(start), err)
	}
	if c.trace && c.logger != nil {
		attrs := []any{"type", typ.String(), "duration", time.Since(start)}
		if err != nil {
			attrs = append(attrs, "error", err.Error())
		}
		c.logger.Info("resolve", attrs...)
	}
}

// call invokes the factory, resolving its parameters from the container.
func (c *Container) call(b *binding, typ reflect.Type, path []reflect.Type) (reflect.Value, error) {
	fv := reflect.ValueOf(b.factory)
	ft := fv.Type()
	in := make([]reflect.Value, 0, ft.NumIn())
	for i := 0; i < ft.NumIn(); i++ {
		dep, err := c.resolve(ft.In(i), append(path, typ))
		if err != nil {
			return reflect.Value{}, err
		}
		in = append(in, dep)
	}
	out := fv.Call(in)
	if len(out) == 2 && !out[1].IsNil() {
		return reflect.Value{}, &ResolveError{
			Type: typ.String(),
			Path: pathNames(append(path, typ)),
			Err:  out[1].Interface().(error),
		}
	}
	return out[0], nil
}

func pathNames(path []reflect.Type) []string {
	names := make([]string, 0, len(path))
	for _, t := range path {
		names = append(names, t.String())
	}
	return names
}
