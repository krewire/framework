package test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// RequestBuilder is a fluent HTTP request builder (KWF-TST-H7P-010).
type RequestBuilder struct {
	t       *testing.T
	method  string
	path    string
	headers http.Header
	cookies []*http.Cookie
	query   url.Values
	form    url.Values
	jsonVal any
	body    io.Reader
	ctx     context.Context
}

// Request creates a new builder with default GET and path "/".
func Request(t *testing.T) *RequestBuilder {
	t.Helper()
	return &RequestBuilder{
		t:       t,
		method:  "GET",
		path:    "/",
		headers: make(http.Header),
		query:   make(url.Values),
		form:    make(url.Values),
	}
}

// GET creates a GET builder for path.
func GET(t *testing.T, path string) *RequestBuilder {
	return Request(t).Method("GET").Path(path)
}

// POST creates a POST builder for path with optional body.
func POST(t *testing.T, path string, body io.Reader) *RequestBuilder {
	b := Request(t).Method("POST").Path(path)
	if body != nil {
		b.body = body
	}
	return b
}

// JSONRequest creates a builder with JSON body and Content-Type.
func JSONRequest(t *testing.T, method, path string, v any) *RequestBuilder {
	return Request(t).Method(method).Path(path).JSON(v)
}

func (b *RequestBuilder) Method(m string) *RequestBuilder {
	b.method = m
	return b
}

func (b *RequestBuilder) Path(p string) *RequestBuilder {
	b.path = p
	return b
}

func (b *RequestBuilder) Header(k, v string) *RequestBuilder {
	b.headers.Add(k, v)
	return b
}

func (b *RequestBuilder) Cookie(c *http.Cookie) *RequestBuilder {
	if c != nil {
		b.cookies = append(b.cookies, c)
	}
	return b
}

func (b *RequestBuilder) Query(k, v string) *RequestBuilder {
	b.query.Add(k, v)
	return b
}

func (b *RequestBuilder) Form(k, v string) *RequestBuilder {
	b.form.Add(k, v)
	return b
}

func (b *RequestBuilder) JSON(v any) *RequestBuilder {
	b.jsonVal = v
	return b
}

func (b *RequestBuilder) Body(r io.Reader) *RequestBuilder {
	b.body = r
	return b
}

func (b *RequestBuilder) WithContext(ctx context.Context) *RequestBuilder {
	b.ctx = ctx
	return b
}

// Request builds the *http.Request, applying query, form, JSON, headers, cookies.
func (b *RequestBuilder) Request() *http.Request {
	b.t.Helper()
	path := b.path
	if len(b.query) > 0 {
		if strings.Contains(path, "?") {
			path += "&" + b.query.Encode()
		} else {
			path += "?" + b.query.Encode()
		}
	}
	var body io.Reader = b.body
	if b.jsonVal != nil {
		data, err := json.Marshal(b.jsonVal)
		if err != nil {
			b.t.Fatalf("Request.JSON: %v", err)
		}
		body = bytes.NewReader(data)
		if b.headers.Get("Content-Type") == "" {
			b.headers.Set("Content-Type", "application/json")
		}
	} else if len(b.form) > 0 {
		encoded := b.form.Encode()
		body = strings.NewReader(encoded)
		if b.headers.Get("Content-Type") == "" {
			b.headers.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	req, err := http.NewRequest(b.method, path, body)
	if err != nil {
		b.t.Fatalf("Request: %v", err)
	}
	for k, vs := range b.headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	for _, c := range b.cookies {
		req.AddCookie(c)
	}
	if b.ctx != nil {
		req = req.WithContext(b.ctx)
	}
	return req
}

// Response is a chainable assertion wrapper around httptest.ResponseRecorder (KWF-TST-H7P-012).
type Response struct {
	t   *testing.T
	rec *httptest.ResponseRecorder
}

// Do executes handler with req and returns a Response for chaining (KWF-TST-H7P-011).
func Do(t *testing.T, handler http.Handler, req *http.Request) *Response {
	t.Helper()
	if h, ok := handler.(http.HandlerFunc); ok {
		_ = h
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return &Response{t: t, rec: rec}
}

// Status asserts rec.Code == want.
func (r *Response) Status(want int) *Response {
	r.t.Helper()
	if r.rec.Code != want {
		r.t.Errorf("Status: got %d want %d body %q", r.rec.Code, want, r.rec.Body.String())
	}
	return r
}

// Header asserts header key == want (first value).
func (r *Response) Header(key, want string) *Response {
	r.t.Helper()
	got := r.rec.Header().Get(key)
	if got != want {
		r.t.Errorf("Header %q: got %q want %q", key, got, want)
	}
	return r
}

// Contains asserts body contains text.
func (r *Response) Contains(text string) *Response {
	r.t.Helper()
	if !strings.Contains(r.rec.Body.String(), text) {
		r.t.Errorf("Contains: body %q does not contain %q", r.rec.Body.String(), text)
	}
	return r
}

// NotContains asserts body does not contain text.
func (r *Response) NotContains(text string) *Response {
	r.t.Helper()
	if strings.Contains(r.rec.Body.String(), text) {
		r.t.Errorf("NotContains: body %q should not contain %q", r.rec.Body.String(), text)
	}
	return r
}

// JSON decodes body as JSON into v, strictly.
func (r *Response) JSON(v any) *Response {
	r.t.Helper()
	if err := json.Unmarshal(r.rec.Body.Bytes(), v); err != nil {
		r.t.Errorf("JSON: decode failed %v body %q", err, r.rec.Body.String())
	}
	return r
}

// Cookie returns the cookie with name, failing if not found.
func (r *Response) Cookie(name string) *http.Cookie {
	r.t.Helper()
	for _, c := range r.rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	r.t.Errorf("Cookie: %q not found, cookies %v", name, r.rec.Result().Cookies())
	return nil
}

// RedirectTo asserts status is 3xx and Location == wantPath.
func (r *Response) RedirectTo(wantPath string) *Response {
	r.t.Helper()
	if r.rec.Code < 300 || r.rec.Code >= 400 {
		r.t.Errorf("RedirectTo: got status %d want 3xx", r.rec.Code)
	}
	got := r.rec.Header().Get("Location")
	if got != wantPath {
		r.t.Errorf("RedirectTo: got Location %q want %q", got, wantPath)
	}
	return r
}

// Body returns the body as string.
func (r *Response) Body() string {
	return r.rec.Body.String()
}

// Recorder returns the underlying recorder for escape hatches.
func (r *Response) Recorder() *httptest.ResponseRecorder {
	return r.rec
}

// Server starts a httptest.Server and registers Cleanup (KWF-TST-H7P-013).
func Server(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// CookieJar is a simple in-memory jar for WithCookies (KWF-TST-H7P-014).
type CookieJar struct {
	cookies []*http.Cookie
}

// NewCookieJar creates an empty jar.
func NewCookieJar() *CookieJar { return &CookieJar{} }

// AddFromResponse adds Set-Cookie cookies from a Response.
func (j *CookieJar) AddFromResponse(r *Response) {
	if r == nil || r.rec == nil {
		return
	}
	j.cookies = append(j.cookies, r.rec.Result().Cookies()...)
}

// Cookies returns the stored cookies.
func (j *CookieJar) Cookies() []*http.Cookie { return j.cookies }

// WithCookies adds jar cookies to req.
func WithCookies(req *http.Request, jar *CookieJar) *http.Request {
	if jar == nil {
		return req
	}
	for _, c := range jar.cookies {
		req.AddCookie(c)
	}
	return req
}
