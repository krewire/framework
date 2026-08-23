package web

import (
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
