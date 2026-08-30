// Tests for KWF-WEB-J7K2P
package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	ftest "github.com/krewire/framework/test"
)

// Spec: KWF-WEB-J7K2P FRK-LRV-001 Scope: Unit
func TestFRK_LRV_001_OptionsHead(t *testing.T) {
	r := NewRouter()
	r.Options("/opts", func(w http.ResponseWriter, _ *http.Request, _ Params) { w.Write([]byte("opts")) })
	r.Head("/head", func(w http.ResponseWriter, _ *http.Request, _ Params) { w.Write([]byte("head")) })
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodOptions, "/opts", nil))
	ftest.EqualStatus(t, rec, http.StatusOK)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, ftest.NewRequest(t, http.MethodHead, "/head", nil))
	ftest.EqualStatus(t, rec2, http.StatusOK)
}

// Spec: KWF-WEB-J7K2P FRK-LRV-002 Scope: Unit
func TestFRK_LRV_002_Any(t *testing.T) {
	r := NewRouter()
	r.Any("/any", func(w http.ResponseWriter, _ *http.Request, _ Params) { w.Write([]byte("any")) })
	for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodHead} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, ftest.NewRequest(t, m, "/any", nil))
		ftest.EqualStatus(t, rec, http.StatusOK)
	}
	if len(r.Routes()) != 7 {
		t.Fatalf("Any should register 7 routes, got %d", len(r.Routes()))
	}
}

// Spec: KWF-WEB-J7K2P FRK-LRV-003 Scope: Unit
func TestFRK_LRV_003_Match(t *testing.T) {
	r := NewRouter()
	r.Match([]string{"GET", "post"}, "/match", func(w http.ResponseWriter, _ *http.Request, _ Params) { w.Write([]byte("m")) })
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/match", nil))
	ftest.EqualStatus(t, rec, http.StatusOK)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, ftest.NewRequest(t, http.MethodPost, "/match", nil))
	ftest.EqualStatus(t, rec2, http.StatusOK)
	rec3 := httptest.NewRecorder()
	r.ServeHTTP(rec3, ftest.NewRequest(t, http.MethodPut, "/match", nil))
	ftest.EqualStatus(t, rec3, http.StatusNotFound)
	r.Match([]string{}, "/empty", func(w http.ResponseWriter, _ *http.Request, _ Params) {})
	if r.HasRoute(http.MethodGet, "/empty") {
		t.Error("empty Match should be no-op")
	}
}

// Spec: KWF-WEB-J7K2P FRK-LRV-004 Scope: Unit
func TestFRK_LRV_004_Fallback(t *testing.T) {
	r := NewRouter()
	r.Get("/exists", func(w http.ResponseWriter, _ *http.Request, _ Params) { w.Write([]byte("ok")) })
	r.Fallback(func(w http.ResponseWriter, _ *http.Request, _ Params) {
		w.WriteHeader(404)
		w.Write([]byte("fallback"))
	})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/missing", nil))
	ftest.EqualStatus(t, rec, http.StatusNotFound)
	if rec.Body.String() != "fallback" {
		t.Errorf("fallback = %q", rec.Body.String())
	}
}

// Spec: KWF-WEB-J7K2P FRK-LRV-005 FRK-LRV-006 Scope: Unit
func TestFRK_LRV_005_Redirect(t *testing.T) {
	r := NewRouter()
	r.Redirect("/old", "/new", http.StatusMovedPermanently)
	r.PermanentRedirect("/old2", "/new2")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/old", nil))
	ftest.EqualStatus(t, rec, http.StatusMovedPermanently)
	if rec.Header().Get("Location") != "/new" {
		t.Error("Location")
	}
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, ftest.NewRequest(t, http.MethodGet, "/old2", nil))
	ftest.EqualStatus(t, rec2, http.StatusMovedPermanently)
	// default 302
	r2 := NewRouter()
	r2.Redirect("/a", "/b")
	rec3 := httptest.NewRecorder()
	r2.ServeHTTP(rec3, ftest.NewRequest(t, http.MethodGet, "/a", nil))
	ftest.EqualStatus(t, rec3, http.StatusFound)
}

// Spec: KWF-WEB-J7K2P FRK-LRV-010 FRK-LRV-011 FRK-LRV-012 Scope: Unit
func TestFRK_LRV_010_PrefixNameMiddleware(t *testing.T) {
	r := NewRouter()
	called := ""
	mw := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			called += "mw"
			next.ServeHTTP(w, req)
		})
	}
	g := r.Prefix("/api").Name("api.").Middleware(mw)
	g.GET("/users").Name("users.index").Handle(func(w http.ResponseWriter, _ *http.Request, _ Params) { w.Write([]byte("ok")) })
	if len(r.Routes()) != 1 || r.Routes()[0].Name != "api.users.index" || r.Routes()[0].Pattern != "/api/users" {
		t.Fatalf("Routes = %v", r.Routes())
	}
	url, err := r.URL("api.users.index", nil)
	if err != nil {
		t.Fatal(err)
	}
	if url != "/api/users" {
		t.Errorf("URL = %q", url)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/api/users", nil))
	ftest.EqualStatus(t, rec, http.StatusOK)
	if called != "mw" {
		t.Error("middleware not applied")
	}
	// As alias
	r2 := NewRouter()
	r2.As("v1.").GET("/x").Name("x").Handle(func(w http.ResponseWriter, _ *http.Request, _ Params) {})
	if r2.Routes()[0].Name != "v1.x" {
		t.Errorf("As alias name = %q", r2.Routes()[0].Name)
	}
}

// Spec: KWF-WEB-J7K2P FRK-LRV-014 Scope: Unit
func TestFRK_LRV_014_Where(t *testing.T) {
	r := NewRouter()
	r.Group("/items").Where("id", "\\d+").GET("/{id}").Name("items.show").Where("id", "\\d+").Handle(func(w http.ResponseWriter, _ *http.Request, _ Params) {})
	infos := r.Routes()
	if len(infos) != 1 || infos[0].Constraints["id"] != "\\d+" {
		t.Fatalf("Where constraints = %v", infos[0].Constraints)
	}
	// group Where
	r2 := NewRouter()
	g := r2.Where("id", "\\d+")
	g.Get("/users/{id}", func(w http.ResponseWriter, _ *http.Request, _ Params) {})
	if g.Routes()[0].Constraints["id"] != "\\d+" {
		t.Error("group Where")
	}
	if !containsStr(r.DebugString(), "api.") && len(r2.DebugString()) == 0 {
		// dummy check to avoid unused
	}
}

// Spec: KWF-WEB-J7K2P FRK-LRV-020 FRK-LRV-021 FRK-LRV-022 Scope: Unit
func TestFRK_LRV_020_ResourceFullAndApi(t *testing.T) {
	r := NewRouter()
	ctrl := ResourceController{
		Index:   func(w http.ResponseWriter, _ *http.Request, _ Params) { w.Write([]byte("index")) },
		Create:  func(w http.ResponseWriter, _ *http.Request, _ Params) { w.Write([]byte("create")) },
		Store:   func(w http.ResponseWriter, _ *http.Request, _ Params) { w.Write([]byte("store")) },
		Show:    func(w http.ResponseWriter, _ *http.Request, _ Params) { w.Write([]byte("show")) },
		Edit:    func(w http.ResponseWriter, _ *http.Request, _ Params) { w.Write([]byte("edit")) },
		Update:  func(w http.ResponseWriter, _ *http.Request, _ Params) { w.Write([]byte("update")) },
		Destroy: func(w http.ResponseWriter, _ *http.Request, _ Params) { w.Write([]byte("destroy")) },
	}
	r.Resource("/photos", ctrl)
	// 7 actions -> Index, Create, Store, Show, Edit, Update(2 routes PUT+PATCH), Destroy = 8 routes (update has 2)
	if len(r.Routes()) != 8 {
		t.Fatalf("Resource should register 8 routes (update has PUT+PATCH), got %d: %v", len(r.Routes()), r.Routes())
	}
	// check names
	names := map[string]bool{}
	for _, ri := range r.Routes() {
		names[ri.Name] = true
	}
	for _, n := range []string{"photos.index", "photos.create", "photos.store", "photos.show", "photos.edit", "photos.update", "photos.destroy"} {
		if !names[n] {
			t.Errorf("missing name %q", n)
		}
	}

	// ApiResource -> 5 actions
	r2 := NewRouter()
	r2.ApiResource("/photos", ctrl)
	if len(r2.Routes()) != 6 {
		// index, store, show, update(2), destroy = 6
		t.Fatalf("ApiResource should register 6 routes (update 2), got %d: %v", len(r2.Routes()), r2.Routes())
	}
	for _, ri := range r2.Routes() {
		if ri.Name == "photos.create" || ri.Name == "photos.edit" {
			t.Errorf("ApiResource should skip create/edit but got %q", ri.Name)
		}
	}
}

// Spec: KWF-WEB-J7K2P FRK-LRV-023 Scope: Unit
func TestFRK_LRV_023_OnlyExcept(t *testing.T) {
	r := NewRouter()
	ctrl := ResourceController{
		Index:   func(w http.ResponseWriter, _ *http.Request, _ Params) {},
		Store:   func(w http.ResponseWriter, _ *http.Request, _ Params) {},
		Show:    func(w http.ResponseWriter, _ *http.Request, _ Params) {},
		Update:  func(w http.ResponseWriter, _ *http.Request, _ Params) {},
		Destroy: func(w http.ResponseWriter, _ *http.Request, _ Params) {},
		Create:  func(w http.ResponseWriter, _ *http.Request, _ Params) {},
		Edit:    func(w http.ResponseWriter, _ *http.Request, _ Params) {},
	}
	r.ResourceFiltered("/photos", ctrl, ResourceOptions{Only: []string{"index", "show"}})
	if len(r.Routes()) != 2 {
		t.Fatalf("Only index,show => 2 routes, got %d", len(r.Routes()))
	}
	r2 := NewRouter()
	r2.ResourceFiltered("/photos", ctrl, ResourceOptions{Except: []string{"create", "edit"}})
	// should skip create/edit -> 6 routes (as ApiResource)
	if len(r2.Routes()) != 6 {
		t.Fatalf("Except create,edit => 6 routes, got %d: %v", len(r2.Routes()), r2.Routes())
	}
}

// Spec: KWF-WEB-J7K2P FRK-LRV-030 FRK-LRV-031 Scope: Unit
func TestFRK_LRV_030_BaseController(t *testing.T) {
	type UserController struct {
		BaseController
	}
	ctrl := &UserController{}
	ctrl.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Header().Set("X-Ctrl", "1"); next.ServeHTTP(w, r) })
	})
	if len(ctrl.Middleware()) != 1 {
		t.Fatal("Middleware")
	}
	r := NewRouter()
	r.Group("/users", ctrl.Middleware()...).Register(&testController{msg: "hi"})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/users/ping", nil))
	if rec.Header().Get("X-Ctrl") != "1" {
		t.Error("BaseController middleware not applied")
	}
	// helpers
	bc := &BaseController{}
	if bc.OK(map[string]string{"a": "b"}) == nil {
		t.Error("OK")
	}
	if bc.Created(map[string]string{"a": "b"}) == nil {
		t.Error("Created")
	}
	if bc.NoContent() == nil {
		t.Error("NoContent")
	}
	if bc.BadRequest("x") == nil {
		t.Error("BadRequest")
	}
	if bc.NotFoundErr("x") == nil {
		t.Error("NotFoundErr")
	}
}

type testController struct{ msg string }

func (c *testController) Register(r *Router) {
	r.Get("/ping", func(w http.ResponseWriter, _ *http.Request, _ Params) { w.Write([]byte(c.msg)) })
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOfStr(s, sub) >= 0)
}
func indexOfStr(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
