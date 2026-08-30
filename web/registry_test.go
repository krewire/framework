// Tests for KWF-WEB-Q8T2R
package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ftest "github.com/krewire/framework/test"
)

// Spec: KWF-WEB-Q8T2R FRK-REG-003 Scope: Unit
func TestFRK_REG_003_ModuleMounting(t *testing.T) {
	r := NewRouter()
	mod := Module{
		Prefix: "/catalog",
		Register: func(g *Router) {
			g.Get("/items", func(w http.ResponseWriter, req *http.Request, p Params) {
				w.Write([]byte("catalog items"))
			})
			g.Get("/items/{id}", func(w http.ResponseWriter, req *http.Request, p Params) {
				w.Write([]byte(p["id"]))
			})
		},
	}
	r.Register(mod)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/catalog/items", nil))
	ftest.EqualStatus(t, rec, http.StatusOK)
	ftest.Equal(t, "catalog items", rec.Body.String())

	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, ftest.NewRequest(t, http.MethodGet, "/catalog/items/42", nil))
	ftest.EqualStatus(t, rec2, http.StatusOK)
	ftest.Equal(t, "42", rec2.Body.String())

	infos := r.Routes()
	if len(infos) != 2 {
		t.Fatalf("Routes len = %d want 2", len(infos))
	}
	if infos[0].Pattern != "/catalog/items" || infos[1].Pattern != "/catalog/items/{id}" {
		t.Errorf("patterns = %v", infos)
	}
}

// Spec: KWF-WEB-Q8T2R FRK-REG-004 Scope: Unit
func TestFRK_REG_004_MountShorthand(t *testing.T) {
	r := NewRouter()
	r.Mount("/api", func(api *Router) {
		api.Get("/ping", func(w http.ResponseWriter, req *http.Request, p Params) {
			w.Write([]byte("pong"))
		})
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/api/ping", nil))
	ftest.EqualStatus(t, rec, http.StatusOK)
	ftest.Equal(t, "pong", rec.Body.String())
}

// Spec: KWF-WEB-Q8T2R FRK-REG-011 FRK-REG-012 Scope: Unit
func TestFRK_REG_011_RoutesAndHasRoute(t *testing.T) {
	r := NewRouter()
	r.Get("/users/{id}", func(w http.ResponseWriter, req *http.Request, p Params) {})
	r.Post("/users", func(w http.ResponseWriter, req *http.Request, p Params) {})

	if !r.HasRoute(http.MethodGet, "/users/{id}") {
		t.Error("HasRoute should be true for GET /users/{id}")
	}
	if r.HasRoute(http.MethodDelete, "/users/{id}") {
		t.Error("HasRoute should be false for DELETE")
	}
	if !r.RouteExists(http.MethodPost, "/users") {
		t.Error("RouteExists should be true for POST /users")
	}
	infos := r.Routes()
	if len(infos) != 2 {
		t.Fatalf("Routes len %d", len(infos))
	}
	if infos[0].Method != http.MethodGet || infos[0].Pattern != "/users/{id}" {
		t.Errorf("first route = %v", infos[0])
	}
}

// Spec: KWF-WEB-Q8T2R FRK-REG-014 Scope: Unit
func TestFRK_REG_014_DuplicateDetection(t *testing.T) {
	r := NewRouter()
	r.Get("/dup", func(w http.ResponseWriter, req *http.Request, p Params) {})
	r.Get("/dup", func(w http.ResponseWriter, req *http.Request, p Params) {})
	dups := r.CheckDuplicates()
	if len(dups) != 1 {
		t.Fatalf("dups len %d want 1", len(dups))
	}
	if dups[0].Pattern != "/dup" || dups[0].Method != http.MethodGet {
		t.Errorf("dup = %v", dups[0])
	}
	if err := r.Validate(); err == nil {
		t.Error("Validate should error on duplicate")
	}
	r2 := NewRouter()
	r2.Get("/a", func(w http.ResponseWriter, req *http.Request, p Params) {})
	if err := r2.Validate(); err != nil {
		t.Errorf("Validate clean = %v", err)
	}
}

// Spec: KWF-WEB-Q8T2R FRK-REG-014 MustHandle Scope: Unit
func TestFRK_REG_014_MustHandlePanicsOnDuplicate(t *testing.T) {
	r := NewRouter()
	r.MustHandle(http.MethodGet, "/x", func(w http.ResponseWriter, req *http.Request, p Params) {})
	defer func() {
		if rec := recover(); rec == nil {
			t.Error("MustHandle should panic on duplicate")
		}
	}()
	r.MustHandle(http.MethodGet, "/x", func(w http.ResponseWriter, req *http.Request, p Params) {})
}

// Spec: KWF-WEB-Q8T2R FRK-REG-015 Scope: Unit
func TestFRK_REG_015_DebugString(t *testing.T) {
	r := NewRouter()
	r.GET("/b").Name("b").Handle(func(w http.ResponseWriter, req *http.Request, p Params) {})
	r.POST("/a").Handle(func(w http.ResponseWriter, req *http.Request, p Params) {})
	ds := r.DebugString()
	if !strings.Contains(ds, "GET") || !strings.Contains(ds, "POST") {
		t.Errorf("DebugString = %q", ds)
	}
	if !strings.Contains(ds, "b") {
		t.Errorf("DebugString should contain route name b: %q", ds)
	}
	// Deterministic: GET before POST lexicographically? Actually GET < POST
	lines := strings.Split(strings.TrimSpace(ds), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines %v", lines)
	}
	if !strings.HasPrefix(lines[0], "GET") {
		t.Errorf("first line should be GET: %q", lines[0])
	}
}

// Spec: KWF-WEB-Q8T2R FRK-REG-021 FRK-REG-022 Scope: Unit
func TestFRK_REG_021_Resource(t *testing.T) {
	r := NewRouter()
	r.Resource("/users", ResourceController{
		Index: func(w http.ResponseWriter, req *http.Request, p Params) { w.Write([]byte("index")) },
		Show:  func(w http.ResponseWriter, req *http.Request, p Params) { w.Write([]byte("show:" + p["id"])) },
	})
	if len(r.Routes()) != 2 {
		t.Fatalf("Resource with 2 handlers should create 2 routes, got %d", len(r.Routes()))
	}
	url, err := r.URL("users.show", PathParams{"id": "7"})
	if err != nil {
		t.Fatal(err)
	}
	ftest.Equal(t, "/users/7", url)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, url, nil))
	ftest.EqualStatus(t, rec, http.StatusOK)
	ftest.Equal(t, "show:7", rec.Body.String())

	// Group-scoped resource: /api/users
	r2 := NewRouter()
	api := r2.Group("/api")
	api.Resource("/users", ResourceController{
		Index: func(w http.ResponseWriter, req *http.Request, p Params) { w.Write([]byte("api index")) },
	})
	url2, err := r2.URL("api.users.index", nil)
	if err != nil {
		t.Fatal(err)
	}
	ftest.Equal(t, "/api/users", url2)
	rec2 := httptest.NewRecorder()
	r2.ServeHTTP(rec2, ftest.NewRequest(t, http.MethodGet, url2, nil))
	ftest.EqualStatus(t, rec2, http.StatusOK)
}

// Spec: KWF-WEB-Q8T2R FRK-REG-002 FRK-REG-031 Scope: Unit — modular monolith with 3 modules
func TestFRK_REG_031_ModularMonolithThreeModules(t *testing.T) {
	r := NewRouter()

	catalogMod := Module{Prefix: "/catalog", Register: func(g *Router) {
		g.Get("/products", func(w http.ResponseWriter, req *http.Request, p Params) { w.Write([]byte("catalog")) })
	}}
	orderMod := Module{Prefix: "/order", Register: func(g *Router) {
		g.Post("/checkout", func(w http.ResponseWriter, req *http.Request, p Params) { w.Write([]byte("order")) })
	}}
	userMod := Module{Prefix: "/user", Register: func(g *Router) {
		g.Get("/me", func(w http.ResponseWriter, req *http.Request, p Params) { w.Write([]byte("user")) })
	}}
	r.Register(catalogMod)
	r.Register(orderMod)
	r.Register(userMod)

	if len(r.Routes()) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(r.Routes()))
	}
	for _, tc := range []struct{ method, path, want string }{
		{http.MethodGet, "/catalog/products", "catalog"},
		{http.MethodPost, "/order/checkout", "order"},
		{http.MethodGet, "/user/me", "user"},
	} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, ftest.NewRequest(t, tc.method, tc.path, nil))
		ftest.EqualStatus(t, rec, http.StatusOK)
		ftest.Equal(t, tc.want, rec.Body.String())
	}
	// No duplicates
	if dups := r.CheckDuplicates(); len(dups) != 0 {
		t.Errorf("unexpected duplicates %v", dups)
	}
}

// Spec: KWF-WEB-Q8T2R FRK-REG-002 Registrar func alias
func TestFRK_REG_002_AsRegistrar(t *testing.T) {
	r := NewRouter()
	reg := AsRegistrar(func(rr *Router) {
		rr.Get("/alias", func(w http.ResponseWriter, req *http.Request, p Params) { w.Write([]byte("alias")) })
	})
	r.Register(reg)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/alias", nil))
	ftest.EqualStatus(t, rec, http.StatusOK)
}
