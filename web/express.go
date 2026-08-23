package web

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"

	"github.com/krewire/libs/validate"
)

// Request wraps the raw request with route params and binding helpers.
// Obtain one inside generic handlers (H/HQ) or build with Wrap.
type Request struct {
	*http.Request
	Params Params
}

// Wrap pairs req with matched route params.
func Wrap(req *http.Request, p Params) *Request {
	return &Request{Request: req, Params: p}
}

// Param returns the path parameter key, empty when absent.
func (r *Request) Param(key string) string {
	return r.Params[key]
}

// Query returns the first value of the query parameter key.
func (r *Request) Query(key string) string {
	return r.URL.Query().Get(key)
}

// Bind decodes the JSON body into dst and validates it.
func (r *Request) Bind(dst any) error {
	return DecodeAndValidate(r.Request, dst)
}

// BindQuery maps query parameters onto dst fields tagged `query:"name"`,
// then validates dst. Supported kinds: string, *string, signed/unsigned
// ints, floats, bool, and []string. Unknown query keys are ignored; absent
// keys leave fields untouched.
func (r *Request) BindQuery(dst any) error {
	values := r.URL.Query()
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return BadRequest("query bind target must be a struct pointer")
	}
	elem := v.Elem()
	t := elem.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name := f.Tag.Get("query")
		if name == "" || name == "-" {
			continue
		}
		field := elem.Field(i)
		vs, ok := values[name]
		if !ok || len(vs) == 0 {
			continue
		}
		if err := setFromQuery(field, vs); err != nil {
			return BadRequest("invalid query parameter " + name)
		}
	}
	return validateStruct(dst)
}

func setFromQuery(field reflect.Value, vs []string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(vs[0])
	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			out := reflect.MakeSlice(field.Type(), len(vs), len(vs))
			for i, s := range vs {
				out.Index(i).SetString(s)
			}
			field.Set(out)
		}
	case reflect.Pointer:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setFromQuery(field.Elem(), vs)
	case reflect.Bool:
		b, err := strconv.ParseBool(vs[0])
		if err != nil {
			return err
		}
		field.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(vs[0], 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(vs[0], 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(n)
	case reflect.Float32, reflect.Float64:
		fl, err := strconv.ParseFloat(vs[0], field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(fl)
	default:
		return errUnsupportedKind{kind: field.Kind()}
	}
	return nil
}

type errUnsupportedKind struct{ kind reflect.Kind }

func (e errUnsupportedKind) Error() string {
	return "unsupported query field kind " + e.kind.String()
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

// HTML sets an HTML body.
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

// H wraps fn into a HandlerFunc with JSON-body binding:
//
//	r.Post("/users", web.H(func(r *web.Request, in *CreateUser) (any, error) {
//	    return store.Create(in), nil
//	}))
//
// The body binds through Request.Bind (JSON + validate); validation errors
// become 400 envelopes. Returning *Response writes it verbatim; any other
// non-nil result writes as JSON with the builder's status (default 200).
func H[Q any](fn func(*Request, *Q) (any, error)) HandlerFunc {
	return bindHandler(fn, func(r *Request, q *Q) error { return r.Bind(q) })
}

// HQ is H but binds Q from query parameters via Request.BindQuery.
func HQ[Q any](fn func(*Request, *Q) (any, error)) HandlerFunc {
	return bindHandler(fn, func(r *Request, q *Q) error { return r.BindQuery(q) })
}

func bindHandler[Q any](fn func(*Request, *Q) (any, error), bind func(*Request, *Q) error) HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request, p Params) {
		r := Wrap(req, p)
		q := new(Q)
		if err := bind(r, q); err != nil {
			Error(w, err)
			return
		}
		out, err := fn(r, q)
		if err != nil {
			Error(w, err)
			return
		}
		if out == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if rs, ok := out.(*Response); ok {
			rs.Write(w)
			return
		}
		JSON(w, http.StatusOK, out)
	}
}

func validateStruct(dst any) error {
	return validate.Struct(dst)
}
