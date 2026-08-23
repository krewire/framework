// Tests for KWF-WEB-P3V8X
package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ftest "github.com/krewire/framework/test"
)

func mw(tag string, marks *[]string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*marks = append(*marks, tag)
			next.ServeHTTP(w, r)
		})
	}
}

// Spec: KWF-WEB-P3V8X FRK-WEX-002 Scope: Package
func TestFRK_WEX_002_GroupScopedMiddleware(t *testing.T) {
	var marks []string
	r := NewRouter()
	r.Use(mw("global", &marks))

	api := r.Group("/api", mw("api", &marks))
	api.Get("/users/{id}", func(w http.ResponseWriter, req *http.Request, p Params) {
		marks = append(marks, "handler:"+p["id"])
	})
	other := r.Group("/other")
	other.Get("/x", func(w http.ResponseWriter, req *http.Request, p Params) {
		marks = append(marks, "other")
	})

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/api/users/7", nil))
	ftest.EqualStatus(t, rec, http.StatusOK)
	if len(marks) != 3 || marks[0] != "global" || marks[1] != "api" || marks[2] != "handler:7" {
		t.Errorf("scoped chain = %v", marks)
	}

	marks = nil
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, ftest.NewRequest(t, http.MethodGet, "/other/x", nil))
	if len(marks) != 2 || marks[0] != "global" || marks[1] != "other" {
		t.Errorf("sibling must not inherit api middleware: %v", marks)
	}
}

// Spec: KWF-WEB-P3V8X FRK-WEX-001+003 Scope: Package
func TestFRK_WEX_001_RouteBuilder_NameAndURL(t *testing.T) {
	r := NewRouter()
	called := ""
	r.GET("/users/{id}").
		Name("users.show").
		Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				called += "mw"
				next.ServeHTTP(w, req)
			})
		}).
		Handle(func(w http.ResponseWriter, req *http.Request, p Params) {
			called += ":" + p["id"]
		})

	url, err := r.URL("users.show", PathParams{"id": "42"})
	if err != nil {
		t.Fatal(err)
	}
	ftest.Equal(t, "/users/42", url)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, url, nil))
	if called != "mw:42" {
		t.Errorf("route middleware/handler order = %q", called)
	}

	if _, err := r.URL("nope", nil); err == nil {
		t.Error("unknown name should error")
	}
	if _, err := r.URL("users.show", PathParams{}); err == nil {
		t.Error("missing param should error")
	}
}

// Spec: KWF-WEB-P3V8X FRK-WEX-004 Scope: Package
func TestFRK_WEX_004_ControllerRegistration(t *testing.T) {
	r := NewRouter()
	r.Register(&pingController{})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/ping", nil))
	ftest.EqualStatus(t, rec, http.StatusOK)
	ftest.Equal(t, "pong", rec.Body.String())
}

type pingController struct{}

func (c *pingController) Register(r *Router) {
	r.Get("/ping", func(w http.ResponseWriter, req *http.Request, p Params) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})
}

type listQuery struct {
	Page int      `query:"page" validate:"required"`
	Tags []string `query:"tag"`
	OK   bool     `query:"ok"`
}

// Spec: KWF-WEB-P3V8X FRK-WEX-021 Scope: Package
func TestFRK_WEX_021_HQ_QueryBinding(t *testing.T) {
	r := NewRouter()
	r.GET("/list").Handle(HQ(func(req *Request, q *listQuery) (any, error) {
		return map[string]any{"page": q.Page, "tags": q.Tags, "ok": q.OK}, nil
	}))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, ftest.NewRequest(t, http.MethodGet, "/list?page=2&tag=a&tag=b&ok=true", nil))
	ftest.EqualStatus(t, rec, http.StatusOK)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out["page"] != float64(2) {
		t.Errorf("page = %v", out["page"])
	}
	if tags, _ := out["tags"].([]any); len(tags) != 2 {
		t.Errorf("tags = %v", out["tags"])
	}

	// invalid int -> 400 envelope
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, ftest.NewRequest(t, http.MethodGet, "/list?page=abc", nil))
	ftest.EqualStatus(t, rec2, http.StatusBadRequest)
}

type createIn struct {
	Name string `json:"name" validate:"required"`
}

// Spec: KWF-WEB-P3V8X FRK-WEX-010+020 Scope: Package
func TestFRK_WEX_020_H_BodyBindingAndResponseBuilder(t *testing.T) {
	r := NewRouter()
	r.POST("/items").Handle(H(func(req *Request, in *createIn) (any, error) {
		if req.Query("dry") == "1" {
			return NoContent(), nil
		}
		return Created(map[string]string{"name": in.Name, "id": req.Param("pid")}), nil
	}))
	// wrap with a param route to exercise Param access
	r2 := NewRouter()
	r2.Route(http.MethodPost, "/p/{pid}").Handle(H(func(req *Request, in *createIn) (any, error) {
		return Created(map[string]string{"pid": req.Param("pid"), "name": in.Name}), nil
	}))

	rec := httptest.NewRecorder()
	body := strings.NewReader(`{"name":"kiw"}`)
	req := ftest.NewRequest(t, http.MethodPost, "/p/9", body)
	req.Header.Set("Content-Type", "application/json")
	r2.ServeHTTP(rec, req)
	ftest.EqualStatus(t, rec, http.StatusCreated)
	var out map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	ftest.Equal(t, "9", out["pid"])
	ftest.Equal(t, "kiw", out["name"])

	// validation failure -> 400
	rec2 := httptest.NewRecorder()
	bad := ftest.NewRequest(t, http.MethodPost, "/p/9", strings.NewReader(`{}`))
	bad.Header.Set("Content-Type", "application/json")
	r2.ServeHTTP(rec2, bad)
	ftest.EqualStatus(t, rec2, http.StatusBadRequest)

	// NoContent path via H returning *Response
	r3 := NewRouter()
	r3.POST("/n").Handle(H(func(req *Request, in *createIn) (any, error) { return NoContent(), nil }))
	rec3 := httptest.NewRecorder()
	nb := ftest.NewRequest(t, http.MethodPost, "/n", strings.NewReader(`{"name":"x"}`))
	nb.Header.Set("Content-Type", "application/json")
	r3.ServeHTTP(rec3, nb)
	ftest.EqualStatus(t, rec3, http.StatusNoContent)
}

// Spec: KWF-WEB-P3V8X FRK-WEX-011 Scope: Package
func TestFRK_WEX_011_ResponseBuilderVariants(t *testing.T) {
	// Redirect
	rs := Respond().Redirect("/login", http.StatusSeeOther)
	rec := httptest.NewRecorder()
	rs.Write(rec)
	ftest.EqualStatus(t, rec, http.StatusSeeOther)
	ftest.Equal(t, "/login", rec.Header().Get("Location"))

	// Text + custom header
	rec2 := httptest.NewRecorder()
	Respond().Set("X-Kiw", "1").Text("hi").Write(rec2)
	ftest.Equal(t, "1", rec2.Header().Get("X-Kiw"))
	ftest.Equal(t, "hi", rec2.Body.String())

	// Blob
	rec3 := httptest.NewRecorder()
	Respond().Blob("application/octet-stream", []byte{1, 2}).Write(rec3)
	if rec3.Body.Len() != 2 {
		t.Errorf("blob len = %d", rec3.Body.Len())
	}
}
