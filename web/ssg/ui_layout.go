package ssg

import (
	"fmt"
	"html/template"

	"github.com/krewire/framework/ui"
)

// LayoutFromUI converts a reusable ui.Layout shell into an ssg Layout
// template. The ui layout's Title is substituted with {{ .Title }} and its
// Main region with {{ .Content }}, so pages render their root component into
// the shell. This lets Go programs compose page shells from framework/ui
// components instead of inlining layout HTML into a config.
func LayoutFromUI(name string, l ui.Layout) (Layout, error) {
	l.Title = "{{ .Title }}"
	l.Main = template.HTML("{{ .Content }}")
	body := string(l.Render())
	if _, err := template.New(name).Parse(body); err != nil {
		return Layout{}, fmt.Errorf("ssg: layout %q from ui: %w", name, err)
	}
	return Layout{Name: name, Body: body}, nil
}
