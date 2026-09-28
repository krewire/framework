package web

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

type reqIDKey struct{}

// HeaderRequestID is the default HTTP header used for distributed request tracing.
const HeaderRequestID = "X-Request-ID"

// RequestID generates or propagates an X-Request-ID header and stores it in the request context.
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			id := req.Header.Get(HeaderRequestID)
			if id == "" {
				var b [16]byte
				if _, err := rand.Read(b[:]); err == nil {
					id = hex.EncodeToString(b[:])
				} else {
					id = fmt.Sprintf("%d", time.Now().UnixNano())
				}
			}
			ctx := context.WithValue(req.Context(), reqIDKey{}, id)
			w.Header().Set(HeaderRequestID, id)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	}
}

// GetRequestID returns the request ID from the request context or header.
func GetRequestID(req *http.Request) string {
	if req == nil {
		return ""
	}
	if id, ok := req.Context().Value(reqIDKey{}).(string); ok && id != "" {
		return id
	}
	return req.Header.Get(HeaderRequestID)
}

// accessLogRecorder captures response status and bytes written.
type accessLogRecorder struct {
	http.ResponseWriter
	status       int
	bytesWritten int64
	wroteHeader  bool
}

func (r *accessLogRecorder) WriteHeader(status int) {
	if !r.wroteHeader {
		r.status = status
		r.wroteHeader = true
		r.ResponseWriter.WriteHeader(status)
	}
}

func (r *accessLogRecorder) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytesWritten += int64(n)
	return n, err
}

// AccessLogOptions configures the AccessLog middleware.
type AccessLogOptions struct {
	Logger *slog.Logger
}

// AccessLogOption modifies AccessLogOptions.
type AccessLogOption func(*AccessLogOptions)

// WithAccessLogger sets the logger to use for access logs.
func WithAccessLogger(l *slog.Logger) AccessLogOption {
	return func(o *AccessLogOptions) {
		if l != nil {
			o.Logger = l
		}
	}
}

// AccessLog returns middleware that logs completed HTTP requests with latency, status, and request ID.
func AccessLog(opts ...AccessLogOption) Middleware {
	cfg := AccessLogOptions{
		Logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			rec := &accessLogRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, req)

			duration := time.Since(start)
			reqID := GetRequestID(req)
			clientIP := clientIP(req)

			cfg.Logger.Info("http request",
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Int("status", rec.status),
				slog.Duration("duration", duration),
				slog.Int64("bytes", rec.bytesWritten),
				slog.String("client_ip", clientIP),
				slog.String("request_id", reqID),
			)
		})
	}
}

// Recover returns middleware that catches panics in HTTP handlers, logs the stack trace,
// and returns a clean 500 Internal Server Error without leaking internal stack frames.
func Recover(logger ...*slog.Logger) Middleware {
	l := slog.Default()
	if len(logger) > 0 && logger[0] != nil {
		l = logger[0]
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					stack := string(debug.Stack())
					reqID := GetRequestID(req)

					l.Error("web: panic recovered",
						slog.Any("error", rvr),
						slog.String("request_id", reqID),
						slog.String("path", req.URL.Path),
						slog.String("method", req.Method),
						slog.String("stack", stack),
					)

					if strings.Contains(req.Header.Get("Accept"), "application/json") {
						w.Header().Set("Content-Type", "application/json; charset=utf-8")
						w.WriteHeader(http.StatusInternalServerError)
						_ = json.NewEncoder(w).Encode(map[string]string{
							"error":      "Internal Server Error",
							"request_id": reqID,
						})
					} else {
						w.Header().Set("Content-Type", "text/plain; charset=utf-8")
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = fmt.Fprintf(w, "Internal Server Error (ID: %s)\n", reqID)
					}
				}
			}()
			next.ServeHTTP(w, req)
		})
	}
}

func clientIP(req *http.Request) string {
	if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xrip := req.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}
