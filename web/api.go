package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/krewire/libs/validate"
)

// maxBodyBytes bounds request bodies decoded by ReadJSON.
const maxBodyBytes = 1 << 20 // 1 MiB

// Middleware wraps an http.Handler, remaining fully compatible with the
// standard library's middleware conventions.
type Middleware func(next http.Handler) http.Handler

// MiddlewareChain composes middleware left-to-right: the first middleware is
// the outermost wrapper around final.
func MiddlewareChain(mws ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			final = mws[i](final)
		}
		return final
	}
}

// RecoverMiddleware logs panics — together with the recovered goroutine's
// stack (KWL-P8W2N KWF-HTTPV-005) — and turns them into a 500 response,
// keeping internals away from the client. Responses carry a correlation id
// mirrored into the log line so client reports map to server traces
// (KWL-P8W2N KWF-HTTPV-011).
func RecoverMiddleware(logger *slog.Logger) Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					id := correlationID()
					logger.Error("panic recovered",
						"panic", rec,
						"correlation_id", id,
						"stack", callerStack(),
						"method", r.Method,
						"path", r.URL.Path,
					)
					msg := "internal server error"
					if id != "" {
						msg = fmt.Sprintf("internal server error (id %s)", id)
						w.Header().Set("X-Correlation-Id", id)
					}
					Error(w, Internal(msg))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// correlationID returns a short random hex identifier linking an error
// response to its server-side log record.
func correlationID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(b[:])
}

// callerStack renders the current goroutine's call stack, skipping the
// runtime and recovery frames, newest call first.
func callerStack() string {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(3, pcs)
	frames := runtime.CallersFrames(pcs[:n])
	var b strings.Builder
	for {
		f, more := frames.Next()
		if f.Function != "" {
			fmt.Fprintf(&b, "  at %s\n      %s:%d\n", f.Function, f.File, f.Line)
		}
		if !more {
			break
		}
	}
	return b.String()
}

// AccessLogMiddleware logs one structured line per request with method, path,
// status, and duration.
func AccessLogMiddleware(logger *slog.Logger) Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(sw, r)
			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"duration", time.Since(start).String(),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// ErrInvalidJSON classifies a malformed JSON request body.
var ErrInvalidJSON = errors.New("invalid JSON body")

// JSON writes v as a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ReadJSON decodes the request body into dst with a bounded reader, rejecting
// unknown fields and trailing data. An empty body leaves dst at its zero value
// and returns nil; malformed JSON returns an error wrapping ErrInvalidJSON.
func ReadJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes+1))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: trailing data after JSON value", ErrInvalidJSON)
	}
	return nil
}

// DecodeAndValidate decodes a JSON body and validates dst through
// libs/validate in one call.
func DecodeAndValidate(r *http.Request, dst any) error {
	if err := ReadJSON(r, dst); err != nil {
		return err
	}
	return validate.Struct(dst)
}

// HTTPError carries an HTTP status for an error response. It implements error
// and works with errors.Is and errors.As.
type HTTPError struct {
	// Status is the HTTP status code.
	Status int
	// Code is a stable machine-readable code (e.g. "not_found").
	Code string
	// Message is the human-readable message.
	Message string
	// Details carries optional structured payload.
	Details any
}

// Error implements the error interface.
func (e *HTTPError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return http.StatusText(e.Status)
}

// Is supports errors.Is matching on status and optional code.
func (e *HTTPError) Is(target error) bool {
	t, ok := target.(*HTTPError)
	if !ok {
		return false
	}
	return e.Status == t.Status && (t.Code == "" || e.Code == t.Code)
}

// BadRequest returns a 400 HTTPError.
func BadRequest(message string) *HTTPError {
	return &HTTPError{Status: http.StatusBadRequest, Code: "bad_request", Message: message}
}

// NotFound returns a 404 HTTPError.
func NotFound(message string) *HTTPError {
	return &HTTPError{Status: http.StatusNotFound, Code: "not_found", Message: message}
}

// Forbidden returns a 403 HTTPError.
func Forbidden(message string) *HTTPError {
	return &HTTPError{Status: http.StatusForbidden, Code: "forbidden", Message: message}
}

// Internal returns a 500 HTTPError.
func Internal(message string) *HTTPError {
	return &HTTPError{Status: http.StatusInternalServerError, Code: "internal_error", Message: message}
}

// Conflict returns a 409 HTTPError.
func Conflict(message string) *HTTPError {
	return &HTTPError{Status: http.StatusConflict, Code: "conflict", Message: message}
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// Error maps err to an HTTP response: validation failures become 400, HTTPError
// uses its own status, and anything else becomes a generic 500.
func Error(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	var he *HTTPError
	if errors.As(err, &he) {
		writeError(w, he.Status, he.Code, he.Message, he.Details)
		return
	}
	var ve *validate.ValidationError
	if errors.As(err, &ve) {
		writeError(w, http.StatusBadRequest, "validation_error", ve.Error(), nil)
		return
	}
	writeError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
}

func writeError(w http.ResponseWriter, status int, code, message string, details any) {
	JSON(w, status, errorResponse{Code: code, Message: message, Details: details})
}
