package web

import (
	"encoding/json"
	"io"
	"net/http"
)

// HTML writes an HTML response with the given status code.
func HTML(w http.ResponseWriter, code int, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	io.WriteString(w, body)
}

// Redirect responds with a redirect to the target URL.
func Redirect(w http.ResponseWriter, r *http.Request, url string, code int) {
	http.Redirect(w, r, url, code)
}

// Response is a fluent HTTP response under construction. Finish with Write,
// or return it from an H handler to have the framework flush it.
type Response struct {
	status int
	header http.Header
	body   any
	raw    []byte
	ctype  string
	loc    string
}

// Respond starts a 200 response.
func Respond() *Response { return &Response{status: http.StatusOK, header: http.Header{}} }

// Created starts a 201 JSON response carrying v.
func Created(v any) *Response { return Respond().Status(http.StatusCreated).JSON(v) }

// NoContent is a 204 response with no body.
func NoContent() *Response { return &Response{status: http.StatusNoContent, header: http.Header{}} }

// Status sets the response status code.
func (rs *Response) Status(code int) *Response {
	rs.status = code
	return rs
}

// Set sets a response header value.
func (rs *Response) Set(key, value string) *Response {
	rs.header.Set(key, value)
	return rs
}

// JSON sets the body, encoded as JSON on Write.
func (rs *Response) JSON(v any) *Response {
	rs.body = v
	rs.ctype = "application/json; charset=utf-8"
	return rs
}

// Text sets a plain-text body.
func (rs *Response) Text(s string) *Response {
	rs.raw = []byte(s)
	rs.ctype = "text/plain; charset=utf-8"
	return rs
}

// HTML sets an HTML body on the Response builder.
func (rs *Response) HTML(s string) *Response {
	rs.raw = []byte(s)
	rs.ctype = "text/html; charset=utf-8"
	return rs
}

// Blob sets raw bytes with an explicit content type.
func (rs *Response) Blob(ctype string, b []byte) *Response {
	rs.raw = b
	rs.ctype = ctype
	return rs
}

// Redirect turns the response into a redirect to url (default 302).
func (rs *Response) Redirect(url string, code ...int) *Response {
	rs.loc = url
	if len(code) > 0 {
		rs.status = code[0]
	} else {
		rs.status = http.StatusFound
	}
	rs.body = nil
	rs.raw = nil
	return rs
}

// Write flushes the response: headers first, then status, then body.
func (rs *Response) Write(w http.ResponseWriter) {
	for k, vs := range rs.header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	if rs.loc != "" {
		w.Header().Set("Location", rs.loc)
		w.WriteHeader(rs.status)
		return
	}
	if rs.ctype != "" {
		w.Header().Set("Content-Type", rs.ctype)
	}
	w.WriteHeader(rs.status)
	switch {
	case rs.body != nil:
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(rs.body)
	case len(rs.raw) > 0:
		_, _ = w.Write(rs.raw)
	}
}
