// Package web provides the Krewire web framework: a routing and rendering
// layer built on the Go standard library's net/http and html/template.
//
// Pages are defined as routes, rendered through templates, and can be served
// live over HTTP or exported to a directory as a complete static website.
//
// File responsibilities: router.go (route registration and dispatch),
// template.go (named template execution), export.go (static site export),
// response.go (response helpers).
package web
