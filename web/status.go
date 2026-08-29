package web

import "net/http"

// Success (2xx)

// OK is 200 JSON.
func OK(v any) *Response { return Respond().Status(http.StatusOK).JSON(v) }

// Created is 201 JSON.
func CreatedJSON(v any) *Response { return Respond().Status(http.StatusCreated).JSON(v) }

// Accepted is 202 JSON.
func Accepted(v any) *Response { return Respond().Status(http.StatusAccepted).JSON(v) }

// NoContent is 204.
func NoContentResponse() *Response { return NoContent() }

// Redirection (3xx)

// MovedPermanently is 301.
func MovedPermanently(url string) *Response {
	return Respond().Redirect(url, http.StatusMovedPermanently)
}

// Found is 302.
func Found(url string) *Response { return Respond().Redirect(url, http.StatusFound) }

// SeeOther is 303.
func SeeOther(url string) *Response { return Respond().Redirect(url, http.StatusSeeOther) }

// NotModified is 304.
func NotModified() *Response { return &Response{status: http.StatusNotModified, header: http.Header{}} }

// TemporaryRedirect is 307.
func TemporaryRedirect(url string) *Response {
	return Respond().Redirect(url, http.StatusTemporaryRedirect)
}

// PermanentRedirect is 308.
func PermanentRedirect(url string) *Response {
	return Respond().Redirect(url, http.StatusPermanentRedirect)
}

// Client errors (4xx)

func PaymentRequired(message string) *HTTPError {
	return &HTTPError{Status: http.StatusPaymentRequired, Code: "payment_required", Message: message}
}

func MethodNotAllowed(message string) *HTTPError {
	return &HTTPError{Status: http.StatusMethodNotAllowed, Code: "method_not_allowed", Message: message}
}

func NotAcceptable(message string) *HTTPError {
	return &HTTPError{Status: http.StatusNotAcceptable, Code: "not_acceptable", Message: message}
}

func ProxyAuthRequired(message string) *HTTPError {
	return &HTTPError{Status: http.StatusProxyAuthRequired, Code: "proxy_auth_required", Message: message}
}

func RequestTimeoutError(message string) *HTTPError {
	return &HTTPError{Status: http.StatusRequestTimeout, Code: "request_timeout", Message: message}
}

func Gone(message string) *HTTPError {
	return &HTTPError{Status: http.StatusGone, Code: "gone", Message: message}
}

func LengthRequired(message string) *HTTPError {
	return &HTTPError{Status: http.StatusLengthRequired, Code: "length_required", Message: message}
}

func PreconditionFailed(message string) *HTTPError {
	return &HTTPError{Status: http.StatusPreconditionFailed, Code: "precondition_failed", Message: message}
}

func PayloadTooLarge(message string) *HTTPError {
	return &HTTPError{Status: http.StatusRequestEntityTooLarge, Code: "payload_too_large", Message: message}
}

func URITooLong(message string) *HTTPError {
	return &HTTPError{Status: http.StatusRequestURITooLong, Code: "uri_too_long", Message: message}
}

func UnsupportedMediaType(message string) *HTTPError {
	return &HTTPError{Status: http.StatusUnsupportedMediaType, Code: "unsupported_media_type", Message: message}
}

func UnprocessableEntity(message string) *HTTPError {
	return &HTTPError{Status: http.StatusUnprocessableEntity, Code: "unprocessable_entity", Message: message}
}

func TooManyRequests(message string) *HTTPError {
	return &HTTPError{Status: http.StatusTooManyRequests, Code: "too_many_requests", Message: message}
}

// Server errors (5xx)

func NotImplemented(message string) *HTTPError {
	return &HTTPError{Status: http.StatusNotImplemented, Code: "not_implemented", Message: message}
}

func BadGateway(message string) *HTTPError {
	return &HTTPError{Status: http.StatusBadGateway, Code: "bad_gateway", Message: message}
}

func ServiceUnavailable(message string) *HTTPError {
	return &HTTPError{Status: http.StatusServiceUnavailable, Code: "service_unavailable", Message: message}
}

func GatewayTimeout(message string) *HTTPError {
	return &HTTPError{Status: http.StatusGatewayTimeout, Code: "gateway_timeout", Message: message}
}

// Response shortcuts for errors that directly write JSON envelope.
// These are alternatives to returning HTTPError from H handlers.

func UnauthorizedResponse(msg string) *Response {
	return &Response{status: http.StatusUnauthorized, header: http.Header{}, body: map[string]string{"code": "unauthorized", "message": msg}, ctype: "application/json; charset=utf-8"}
}
