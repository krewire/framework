package web

import (
	"encoding/json"
	"net/http"
)

const (
	// MimeProblemJSON is the standard MIME type for RFC 7807 Problem Details.
	MimeProblemJSON = "application/problem+json; charset=utf-8"
)

// Problem represents an RFC 7807 Problem Details object.
type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
	Errors   any    `json:"errors,omitempty"`
}

// WriteProblem writes an RFC 7807 Problem JSON response to w.
func WriteProblem(w http.ResponseWriter, status int, detail string, opts ...func(*Problem)) {
	p := &Problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	}
	for _, f := range opts {
		f(p)
	}
	w.Header().Set("Content-Type", MimeProblemJSON)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}

// ProblemResponse creates a fluent Response formatted as RFC 7807 Problem JSON.
func ProblemResponse(status int, detail string, opts ...func(*Problem)) *Response {
	p := &Problem{
		Type:   "about:blank",
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	}
	for _, f := range opts {
		f(p)
	}
	rs := Respond().Status(status)
	rs.ctype = MimeProblemJSON
	rs.body = p
	return rs
}
