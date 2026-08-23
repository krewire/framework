package app

import (
	"errors"
	"testing"
)

type db struct {
	dsn string
}

type repo struct {
	d  *db
	id int
}

type svc struct {
	r *repo
}

type A struct{}
type B struct{}

func TestContainerResolve(t *testing.T) {
	c := New()
	if err := Provide[*db](c, func() *db { return &db{dsn: "postgres://x"} }); err != nil {
		t.Fatal(err)
	}
	d, err := Resolve[*db](c)
	if err != nil {
		t.Fatal(err)
	}
	if d.dsn != "postgres://x" {
		t.Fatalf("got %q", d.dsn)
	}
}

func TestProvidesNewPerResolve(t *testing.T) {
	c := New()
	_ = Provide[*db](c, func() *db { return &db{dsn: "x"} })
	a, _ := Resolve[*db](c)
	b, _ := Resolve[*db](c)
	if a == b {
		t.Fatal("Provide must build a fresh value per resolve")
	}
}

func TestSingletonReusesValue(t *testing.T) {
	c := New()
	_ = Singleton[*db](c, func() *db { return &db{dsn: "x"} })
	a, _ := Resolve[*db](c)
	b, _ := Resolve[*db](c)
	if a != b {
		t.Fatal("Singleton must reuse the value")
	}
}

func TestResolveWithDependencies(t *testing.T) {
	c := New()
	_ = Singleton[*db](c, func() *db { return &db{dsn: "postgres://x"} })
	_ = Provide[*repo](c, func(d *db) *repo { return &repo{d: d} })
	_ = Provide[*svc](c, func(r *repo) *svc { return &svc{r: r} })

	s, err := Resolve[*svc](c)
	if err != nil {
		t.Fatal(err)
	}
	if s.r.d.dsn != "postgres://x" {
		t.Fatalf("got %q", s.r.d.dsn)
	}
}

func TestMissingBinding(t *testing.T) {
	c := New()
	_, err := Resolve[*db](c)
	if err == nil {
		t.Fatal("expected error")
	}
	var re *ResolveError
	if !errors.As(err, &re) {
		t.Fatalf("expected ResolveError, got %v", err)
	}
}

func TestCircularDependency(t *testing.T) {
	c := New()
	_ = Provide[*A](c, func(a *B) *A { return &A{} })
	_ = Provide[*B](c, func(b *A) *B { return &B{} })
	_, err := Resolve[*A](c)
	var ce *CycleError
	if !errors.As(err, &ce) {
		t.Fatalf("expected CycleError, got %v", err)
	}
}

func TestFactoryError(t *testing.T) {
	c := New()
	want := errors.New("boom")
	_ = Provide[*db](c, func() (*db, error) { return nil, want })
	_, err := Resolve[*db](c)
	if !errors.Is(err, want) {
		t.Fatalf("expected underlying error, got %v", err)
	}
}

func TestOverrideReplacesBinding(t *testing.T) {
	c := New()
	_ = Provide[*db](c, func() *db { return &db{dsn: "prod"} })
	_ = Override[*db](c, func() *db { return &db{dsn: "test"} })
	d, err := Resolve[*db](c)
	if err != nil {
		t.Fatal(err)
	}
	if d.dsn != "test" {
		t.Fatalf("got %q", d.dsn)
	}
}

func TestLockedAfterResolve(t *testing.T) {
	c := New()
	_ = Provide[*db](c, func() *db { return &db{} })
	_, _ = Resolve[*db](c)
	err := Provide[*repo](c, func() *repo { return &repo{} })
	if err == nil {
		t.Fatal("expected lock error")
	}
	if !errors.Is(err, ErrLocked) {
		t.Fatalf("expected ErrLocked, got %v", err)
	}
}

func TestDeterministicOrder(t *testing.T) {
	c := New()
	_ = Singleton[*db](c, func() *db { return &db{dsn: "x"} })
	_ = Provide[*svc](c, func() *svc { return &svc{} })
	_ = Provide[*repo](c, func() *repo { return &repo{} })
	want := []string{"*app.db", "*app.svc", "*app.repo"}
	got := c.Bindings()
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order mismatch at %d: got %v want %v", i, got, want)
		}
	}
}

type counting struct {
	db *db
}

type countingProvider struct {
	n int
}

func (p *countingProvider) Register(c *Container) error {
	p.n++
	if err := Singleton[*db](c, func() *db { return &db{dsn: "postgres://x"} }); err != nil {
		return err
	}
	return Provide[*counting](c, func(d *db) *counting { return &counting{db: d} })
}

func (p *countingProvider) Boot(c *Container) error {
	p.n++
	_ = c
	return nil
}

func TestProviderAssembly(t *testing.T) {
	p := &countingProvider{}
	_, err := NewApp(p).Build()
	if err != nil {
		t.Fatal(err)
	}
	if p.n != 2 {
		t.Fatalf("register+boot expected, got %d calls", p.n)
	}
}

func TestAppContainerResolution(t *testing.T) {
	p := &countingProvider{}
	c, err := NewApp(p).Build()
	if err != nil {
		t.Fatal(err)
	}
	got, err := Resolve[*counting](c)
	if err != nil {
		t.Fatal(err)
	}
	if got.db == nil {
		t.Fatal("dependency not injected")
	}
}

func TestAppTestOverrides(t *testing.T) {
	c, err := NewApp(&countingProvider{}).Test(func() *db { return &db{dsn: "test"} })
	if err != nil {
		t.Fatal(err)
	}
	got, err := Resolve[*db](c)
	if err != nil {
		t.Fatal(err)
	}
	if got.dsn != "test" {
		t.Fatalf("override not applied: got %q", got.dsn)
	}
	cc, err := Resolve[*counting](c)
	if err != nil {
		t.Fatal(err)
	}
	if cc.db.dsn != "test" {
		t.Fatalf("override must reach dependents: got %q", cc.db.dsn)
	}
}

func TestAppBootError(t *testing.T) {
	_, err := NewApp(&boomProvider{}).Build()
	if err == nil {
		t.Fatal("expected boot error")
	}
	if err.Error() != "app: provider *app.boomProvider: boot: boot failed" {
		t.Fatalf("got %v", err)
	}
}

type boomProvider struct{}

func (p *boomProvider) Register(c *Container) error { return nil }
func (p *boomProvider) Boot(c *Container) error     { return errors.New("boot failed") }
