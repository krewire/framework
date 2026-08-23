package web

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func do(r http.Handler, method, path string, body string) *httptest.ResponseRecorder {
	var rd *bytes.Reader
	if body != "" {
		rd = bytes.NewReader([]byte(body))
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rd)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestMiddlewareAppliedToRoutes(t *testing.T) {
	r := NewRouter()
	var calls []string
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			calls = append(calls, "outer")
			next.ServeHTTP(w, req)
		})
	})
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			calls = append(calls, "inner")
			next.ServeHTTP(w, req)
		})
	})
	r.Get("/ping", func(w http.ResponseWriter, _ *http.Request, _ Params) { HTML(w, 200, "pong") })

	rec := do(r, http.MethodGet, "/ping", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	got := strings.Join(calls, ",")
	if got != "outer,inner" {
		t.Errorf("middleware order = %q, want outer,inner", got)
	}
}

func TestMiddlewareChainComposesLeftToRight(t *testing.T) {
	var calls []string
	mw := MiddlewareChain(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, "a")
				next.ServeHTTP(w, r)
			})
		},
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, "b")
				next.ServeHTTP(w, r)
			})
		},
	)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls = append(calls, "h") }))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if got := strings.Join(calls, ","); got != "a,b,h" {
		t.Errorf("chain order = %q, want a,b,h", got)
	}
}

func TestRecoverMiddlewareReturns500(t *testing.T) {
	r := NewRouter()
	r.Use(RecoverMiddleware(nil))
	r.Get("/panic", func(http.ResponseWriter, *http.Request, Params) { panic("boom") })

	rec := do(r, http.MethodGet, "/panic", "")
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "boom") {
		t.Errorf("stack must not leak to client: %q", rec.Body.String())
	}
}

func TestGroupMountedUnderPrefix(t *testing.T) {
	r := NewRouter()
	api := r.Group("/api/v1")
	api.Get("/users/{id}", func(w http.ResponseWriter, _ *http.Request, p Params) {
		JSON(w, http.StatusOK, map[string]string{"id": p["id"]})
	})

	rec := do(r, http.MethodGet, "/api/v1/users/42", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"id":"42"`) {
		t.Errorf("body = %q", rec.Body.String())
	}
	rec = do(r, http.MethodGet, "/api/v1/users/42/extra", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("wrong path must 404, got %d", rec.Code)
	}
}

func TestReadJSONAndDecode(t *testing.T) {
	type DTO struct {
		Name  string `json:"name" validate:"required"`
		Count int    `json:"count" validate:"min=1"`
	}

	var dto DTO
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"krewire","count":2}`))
	if err := DecodeAndValidate(req, &dto); err != nil {
		t.Fatalf("DecodeAndValidate valid: %v", err)
	}
	if dto.Name != "krewire" || dto.Count != 2 {
		t.Errorf("dto = %+v", dto)
	}
}

func TestReadJSONErrors(t *testing.T) {
	var dst map[string]any

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":`))
	if err := ReadJSON(req, &dst); !errors.Is(err, ErrInvalidJSON) {
		t.Errorf("malformed must wrap ErrInvalidJSON, got %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":1} {"b":2}`))
	if err := ReadJSON(req, &dst); !errors.Is(err, ErrInvalidJSON) {
		t.Errorf("trailing data must wrap ErrInvalidJSON, got %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"unknown":1}`))
	var known struct {
		A int `json:"a"`
	}
	if err := ReadJSON(req, &known); !errors.Is(err, ErrInvalidJSON) {
		t.Errorf("unknown field must wrap ErrInvalidJSON, got %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	if err := ReadJSON(req, &dst); err != nil {
		t.Errorf("empty body must decode into zero dst, got %v", err)
	}
}

func TestDecodeAndValidateRejectsBadInput(t *testing.T) {
	type DTO struct {
		Email string `json:"email" validate:"required,email"`
	}
	var dto DTO
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"email":"nope"}`))
	err := DecodeAndValidate(req, &dto)
	var ve interface{ Error() string }
	if !errors.As(err, &ve) {
		t.Fatalf("want validation error, got %T (%v)", err, err)
	}
}

func TestErrorMapping(t *testing.T) {
	type DTO struct {
		Name string `json:"name" validate:"required"`
	}

	t.Run("validation to 400", func(t *testing.T) {
		r := NewRouter()
		r.Post("/echo", func(w http.ResponseWriter, req *http.Request, _ Params) {
			var dto DTO
			if err := DecodeAndValidate(req, &dto); err != nil {
				Error(w, err)
				return
			}
			JSON(w, http.StatusOK, dto)
		})
		rec := do(r, http.MethodPost, "/echo", `{}`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "validation_error") {
			t.Errorf("body = %q", rec.Body.String())
		}
	})

	t.Run("http error uses its status", func(t *testing.T) {
		r := NewRouter()
		r.Get("/gone", func(w http.ResponseWriter, _ *http.Request, _ Params) {
			Error(w, NotFound("resource gone"))
		})
		rec := do(r, http.MethodGet, "/gone", "")
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"code":"not_found"`) {
			t.Errorf("body = %q", rec.Body.String())
		}
	})

	t.Run("unknown to 500", func(t *testing.T) {
		r := NewRouter()
		r.Get("/boom", func(w http.ResponseWriter, _ *http.Request, _ Params) {
			Error(w, errors.New("kaboom"))
		})
		rec := do(r, http.MethodGet, "/boom", "")
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want 500", rec.Code)
		}
		if strings.Contains(rec.Body.String(), "kaboom") {
			t.Errorf("internal detail must not leak: %q", rec.Body.String())
		}
	})
}

func TestHTTPErrorIs(t *testing.T) {
	got := NotFound("missing")
	if !errors.Is(got, &HTTPError{Status: http.StatusNotFound}) {
		t.Error("errors.Is must match by status")
	}
	if errors.Is(got, BadRequest("")) {
		t.Error("different status must not match")
	}
}

func TestCanonicalHelpers(t *testing.T) {
	tests := []struct {
		err    *HTTPError
		status int
		code   string
	}{
		{BadRequest("x"), 400, "bad_request"},
		{NotFound("x"), 404, "not_found"},
		{Forbidden("x"), 403, "forbidden"},
		{Internal("x"), 500, "internal_error"},
		{Conflict("x"), 409, "conflict"},
	}
	for _, tt := range tests {
		if tt.err.Status != tt.status || tt.err.Code != tt.code {
			t.Errorf("%s: got %d/%s, want %d/%s", tt.code, tt.err.Status, tt.err.Code, tt.status, tt.code)
		}
	}
}
