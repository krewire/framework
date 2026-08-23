package ui

import (
	"html/template"
	"strings"
)

// Layout is a composable page shell with named regions. Regions accept
// pre-rendered template.HTML so callers can compose programmatically or from
// declarative config. Omitted regions don't render their wrapper elements.
type Layout struct {
	Title       string
	Description string
	Head        template.HTML // custom <head> content (fonts, meta, styles)
	BodyAttrs   template.HTML // attributes for <body> (e.g., data-page="home")
	Header      template.HTML
	Main        template.HTML
	Footer      template.HTML
	Aside       template.HTML // sidebar
	Stylesheets []string      // output URLs emitted as <link rel="stylesheet">
	Theme       *Theme
}

// Render emits a complete HTML document. The theme script and palette styles
// are injected into <head>. Regions are wrapped in semantic elements with
// data-ui-component attributes for scoped CSS.
func (l Layout) Render() template.HTML {
	var b strings.Builder

	b.WriteString("<!DOCTYPE html>\n")
	b.WriteString(`<html lang="en"`)
	if l.Theme != nil {
		b.WriteString(` data-theme="auto"`)
	}
	b.WriteString(">\n")
	b.WriteString("<head>\n")
	b.WriteString("  <meta charset=\"utf-8\">\n")
	b.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("  <meta name=\"color-scheme\" content=\"light dark\">\n")
	if l.Title != "" {
		b.WriteString("  <title>" + template.HTMLEscapeString(l.Title) + "</title>\n")
	}
	if l.Description != "" {
		b.WriteString("  <meta name=\"description\" content=\"" + template.HTMLEscapeString(l.Description) + "\">\n")
	}
	if l.Head != "" {
		b.WriteString("  " + string(l.Head) + "\n")
	}
	for _, href := range l.Stylesheets {
		b.WriteString("  <link rel=\"stylesheet\" href=\"" + template.HTMLEscapeString(href) + "\">\n")
	}
	if l.Theme != nil {
		b.WriteString("  " + string(l.Theme.Script()) + "\n")
	}
	b.WriteString("</head>\n")
	b.WriteString("<body")
	if l.BodyAttrs != "" {
		b.WriteString(" ")
		b.WriteString(string(l.BodyAttrs))
	}
	b.WriteString(">\n")

	if l.Header != "" {
		b.WriteString(string(l.Header))
		b.WriteString("\n")
	}

	b.WriteString(`<main data-ui-component="main">`)
	if l.Aside != "" {
		b.WriteString(`<aside data-ui-component="aside">`)
		b.WriteString(string(l.Aside))
		b.WriteString("</aside>\n")
	}
	b.WriteString(string(l.Main))
	b.WriteString("</main>\n")

	if l.Footer != "" {
		b.WriteString(string(l.Footer))
		b.WriteString("\n")
	}

	b.WriteString("</body>\n</html>")

	return template.HTML(b.String())
}

// WithHead returns a copy of the layout with additional head content.
func (l Layout) WithHead(head template.HTML) Layout {
	l.Head = l.Head + head
	return l
}

// WithBodyAttrs returns a copy of the layout with additional body attributes.
func (l Layout) WithBodyAttrs(attrs template.HTML) Layout {
	l.BodyAttrs = l.BodyAttrs + attrs
	return l
}

// WithTheme returns a copy of the layout with a theme attached.
func (l Layout) WithTheme(t *Theme) Layout {
	l.Theme = t
	return l
}
