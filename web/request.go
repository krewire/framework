package web

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"

	"github.com/krewire/libs/validate"
)

// Request is the framework's HTTP request wrapper. It is the foundation for
// all input handling — path params, query, form, JSON, files — and is the
// base for FormRequest and validator integration.
//
// Obtain one inside handlers via Wrap or automatically via H/HQ/HF.
// It embeds *http.Request so all stdlib fields (Header, Cookie, Context) remain
// directly accessible.
type Request struct {
	*http.Request
	Params Params
}

// Wrap pairs req with matched route params.
func Wrap(req *http.Request, p Params) *Request {
	return &Request{Request: req, Params: p}
}

// Param returns the path parameter value for key, or "" if absent.
func (r *Request) Param(key string) string {
	return r.Params[key]
}

// ParamInt parses Param(key) as int, returning 0 and false if missing or invalid.
func (r *Request) ParamInt(key string) (int, bool) {
	s := r.Param(key)
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// Query returns the first query value for key.
func (r *Request) Query(key string) string {
	return r.URL.Query().Get(key)
}

// QueryInt parses query key as int.
func (r *Request) QueryInt(key string) (int, bool) {
	s := r.Query(key)
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

// QueryBool parses query key as bool.
func (r *Request) QueryBool(key string) (bool, bool) {
	s := r.Query(key)
	if s == "" {
		return false, false
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return false, false
	}
	return b, true
}

// HasQuery reports whether query key is present.
func (r *Request) HasQuery(key string) bool {
	_, ok := r.URL.Query()[key]
	return ok
}

// FormValue returns the first form value (POST form or query) for key.
// It calls ParseForm lazily.
func (r *Request) FormValue(key string) string {
	_ = r.ParseForm()
	return r.Request.FormValue(key)
}

// FormFile returns the first file for the given form key.
func (r *Request) FormFile(key string) (multipart.File, *multipart.FileHeader, error) {
	return r.Request.FormFile(key)
}

// WantsJSON reports whether the client prefers JSON (Accept contains application/json).
func (r *Request) WantsJSON() bool {
	accept := r.Header.Get("Accept")
	return accept == "" || contains(accept, "application/json") || contains(accept, "*/*")
}

// IsJSON reports whether the request Content-Type is JSON.
func (r *Request) IsJSON() bool {
	ct := r.Header.Get("Content-Type")
	return contains(ct, "application/json")
}

// IsForm reports whether the request is form-encoded.
func (r *Request) IsForm() bool {
	ct := r.Header.Get("Content-Type")
	return contains(ct, "application/x-www-form-urlencoded") || contains(ct, "multipart/form-data")
}

// MethodIs reports whether the request method equals m.
func (r *Request) MethodIs(m string) bool { return r.Method == m }

// Bind decodes the JSON body into dst and validates it (via libs/validate).
// See DecodeAndValidate for limits (1MiB, DisallowUnknownFields).
func (r *Request) Bind(dst any) error {
	return DecodeAndValidate(r.Request, dst)
}

// BindJSON is an alias for Bind (JSON body).
func (r *Request) BindJSON(dst any) error { return r.Bind(dst) }

// BindQuery maps query parameters onto dst fields tagged `query:"name"`,
// then validates dst. See BindQuery for supported kinds.
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

// BindForm maps form values onto dst fields tagged `form:"name"` then validates.
// It handles JSON tags as fallback for form:"-" compatibility and supports the
// same kinds as BindQuery. File fields (*multipart.FileHeader) are ignored here;
// use FormFile directly.
func (r *Request) BindForm(dst any) error {
	if err := r.ParseForm(); err != nil {
		return BadRequest("invalid form: " + err.Error())
	}
	values := r.PostForm
	if len(values) == 0 {
		values = r.Form
	}
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return BadRequest("form bind target must be a struct pointer")
	}
	elem := v.Elem()
	t := elem.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name := f.Tag.Get("form")
		if name == "" {
			name = f.Tag.Get("query")
		}
		if name == "" || name == "-" {
			continue
		}
		field := elem.Field(i)
		if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
			// skip nested struct pointers for form
			continue
		}
		vs, ok := values[name]
		if !ok || len(vs) == 0 {
			continue
		}
		if err := setFromQuery(field, vs); err != nil {
			return BadRequest("invalid form field " + name)
		}
	}
	return validateStruct(dst)
}

// BindMap decodes JSON body into map and validates via struct dst if provided.
// Helper for dynamic payloads.
func (r *Request) BindMap(dst any) error {
	if r.IsJSON() {
		return r.Bind(dst)
	}
	return r.BindForm(dst)
}

// Validate validates dst using libs/validate (validate tags). It is the
// primitive used by Bind* and is exposed for FormRequest validators.
func (r *Request) Validate(dst any) error {
	if err := validate.Struct(dst); err != nil {
		return err
	}
	return nil
}

// DecodeJSON is a low-level helper that decodes JSON body into dst without validation.
func (r *Request) DecodeJSON(dst any) error {
	return ReadJSON(r.Request, dst)
}

// JSONBody decodes JSON body into dst and returns raw bytes for logging.
func (r *Request) JSONBody(dst any) ([]byte, error) {
	if err := r.Bind(dst); err != nil {
		return nil, err
	}
	b, _ := json.Marshal(dst)
	return b, nil
}

func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// FormRequest is the interface for Laravel-like form requests. Implementors
// provide Authorize and Rules for a reusable, testable request object.
//
// Example:
//
//	type CreateUserRequest struct { web.FormRequest; Name string `form:"name" validate:"required,min=3"` }
//	func (r *CreateUserRequest) Authorize(req *web.Request) bool { return req.Identity().HasRole("admin") }
//	func (r *CreateUserRequest) Rules() map[string]string { return map[string]string{"name":"required"} }
//
// The framework will call Authorize before binding and return 403 if false.
type FormRequest interface {
	Authorize(*Request) bool
}

// ValidateFormRequest runs Authorize and then validates dst. It is the
// foundation for generated FormRequest helpers.
func ValidateFormRequest(r *Request, fr FormRequest, dst any) error {
	if fr != nil && !fr.Authorize(r) {
		return Forbidden("unauthorized")
	}
	return validate.Struct(dst)
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
