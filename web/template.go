package web

import (
	"html/template"
	"io"
	"io/fs"
)

// Templates executes named html/templates parsed from an fs.FS.
type Templates struct {
	set *template.Template
}

// ParseTemplates loads the templates matching pattern from fsys.
func ParseTemplates(fsys fs.FS, pattern string) (*Templates, error) {
	set, err := template.ParseFS(fsys, pattern)
	if err != nil {
		return nil, err
	}
	return &Templates{set: set}, nil
}

// Execute writes the named template with data to w.
func (t *Templates) Execute(w io.Writer, name string, data any) error {
	return t.set.ExecuteTemplate(w, name, data)
}
