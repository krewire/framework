package ssg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"sort"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// renderComponent renders a component with data, marks it used, and injects
// its scope attribute onto the rendered root element.
func (s *Site) renderComponent(name string, data any) (template.HTML, error) {
	if _, ok := s.comps[name]; ok {
		s.markUsed(name)
		var buf bytes.Buffer
		if err := s.set.ExecuteTemplate(&buf, name, data); err != nil {
			return "", err
		}
		return scopeFragment(name, buf.String())
	}
	if s.reg != nil {
		if out, err := s.reg.Render(name, componentProps(data)); err == nil {
			return out, nil
		} else if !strings.HasPrefix(err.Error(), "ui: undefined component") {
			return "", err
		}
	}
	return "", fmt.Errorf("ssg: undefined component %q", name)
}

// componentProps strips framework-injected site-wide control keys from a data
// value before it reaches a typed registry component. Templates need the full
// site data (Title, Theme, Collections, …), but typed components decode props
// strictly, so framework keys must not bubble through as unknown fields.
func componentProps(data any) any {
	m, ok := data.(map[string]any)
	if !ok {
		return data
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		switch k {
		case "Title", "Description", "Theme", "IncludeDrafts", "IncludeFuture", "Collections":
			continue
		default:
			out[k] = v
		}
	}
	return out
}

// renderPage renders a page's root component inside its layout.
func (s *Site) renderPage(p *Page) (string, error) {
	root, err := s.renderComponent(p.Root, p.Data)
	if err != nil {
		return "", err
	}
	if s.emitProps && p.Props != nil {
		root, err = injectDataProps(root, p.Props)
		if err != nil {
			return "", err
		}
	}
	_, ok := s.layouts[p.Layout]
	if !ok {
		return "", fmt.Errorf("ssg: undefined layout %q", p.Layout)
	}
	s.markUsed(p.Layout)
	version := ""
	if m, ok := p.Data.(map[string]any); ok {
		if v, ok := m["Version"].(string); ok {
			version = v
		}
	}
	var buf bytes.Buffer
	if err := s.set.ExecuteTemplate(&buf, p.Layout, LayoutData{
		Title:   p.Title,
		Content: root,
		Data:    p.Data,
		Version: version,
	}); err != nil {
		return "", err
	}
	// Inject the layout scope attribute onto the document root so layout
	// styles can be scoped under [data-kiw-layout="name"].
	scoped, err := scopeDocument(p.Layout, buf.String())
	if err != nil {
		return "", err
	}
	body := string(scoped)
	if len(p.Scripts) > 0 {
		body = injectScripts(body, p.Scripts)
	}
	return body, nil
}

// markUsed records that a component or layout was referenced.
func (s *Site) markUsed(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.used[name] = true
}

// collectedCSS concatenates the scoped styles of every component and layout
// referenced during the last build, in deterministic order.
func (s *Site) collectedCSS() string {
	var out strings.Builder
	names := make([]string, 0, len(s.used))
	for name := range s.used {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if c, ok := s.comps[name]; ok && strings.TrimSpace(c.Style) != "" {
			out.WriteString("/* " + name + " */\n")
			out.WriteString(scopeCSS(`[data-kiw-component="`+name+`"]`, c.Style, true))
			out.WriteString("\n")
			continue
		}
		if l, ok := s.layouts[name]; ok && strings.TrimSpace(l.Style) != "" {
			out.WriteString("/* " + name + " */\n")
			out.WriteString(scopeCSS(`[data-kiw-layout="`+name+`"]`, l.Style, false))
			out.WriteString("\n")
		}
	}
	return out.String()
}

// asset resolves a logical asset path to its fingerprinted URL from the
// build manifest, falling back to a plain /assets/ path (KWF-DR5YU).
func (s *Site) asset(name string) string {
	if u, ok := s.manifest[name]; ok {
		return u
	}
	return assetURL(name)
}

// dict builds a map from alternating key-value pairs for template calls like
// {{component "Card" (dict "Title" "Go" "Desc" "...")}} — frontmatter-free.
func dict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict: odd number of arguments")
	}
	m := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		k, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict: key must be string, got %T", values[i])
		}
		m[k] = values[i+1]
	}
	return m, nil
}

// prepare parses component and layout templates into one template set.
func (s *Site) prepare() error {
	if s.set != nil {
		return nil
	}
	funcs := template.FuncMap{}
	for name, fn := range s.funcs {
		funcs[name] = fn
	}
	funcs["component"] = s.renderComponent
	funcs["mount"] = s.renderMount
	funcs["dict"] = dict
	funcs["asset"] = s.asset
	set := template.New("").Funcs(funcs)
	compNames := make([]string, 0, len(s.comps))
	for name := range s.comps {
		compNames = append(compNames, name)
	}
	sort.Strings(compNames)
	for _, name := range compNames {
		if _, err := set.New(name).Parse(s.comps[name].Body); err != nil {
			return fmt.Errorf("ssg: component %q: %w", name, err)
		}
	}
	layoutNames := make([]string, 0, len(s.layouts))
	for name := range s.layouts {
		layoutNames = append(layoutNames, name)
	}
	sort.Strings(layoutNames)
	for _, name := range layoutNames {
		if _, err := set.New(name).Parse(s.layouts[name].Body); err != nil {
			return fmt.Errorf("ssg: layout %q: %w", name, err)
		}
	}
	s.set = set
	return nil
}

// injectDataProps serializes props to a data-props attribute on the first
// element of a rendered fragment, giving client-side frameworks a stable,
// opt-in hydration payload.
func injectDataProps(fragHTML template.HTML, props any) (template.HTML, error) {
	data, err := json.Marshal(props)
	if err != nil {
		return "", fmt.Errorf("ssg: encode page props: %w", err)
	}
	frag, err := html.ParseFragment(strings.NewReader(string(fragHTML)), &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	})
	if err != nil {
		return "", err
	}
	target := firstElement(frag)
	if target == nil {
		return fragHTML, nil
	}
	target.Attr = append(target.Attr, html.Attribute{Key: "data-props", Val: string(data)})
	var buf bytes.Buffer
	for _, n := range frag {
		if err := html.Render(&buf, n); err != nil {
			return "", err
		}
	}
	return template.HTML(buf.String()), nil
}

// injectScripts inserts <script src> tags before the closing </body> tag,
// the documented mount script slot, which is empty by default.
func injectScripts(body string, scripts []string) string {
	var sb strings.Builder
	for _, src := range scripts {
		sb.WriteString(`<script src="`)
		sb.WriteString(src)
		sb.WriteString(`"></script>`)
	}
	sb.WriteString("</body>")
	idx := strings.LastIndex(body, "</body>")
	if idx < 0 {
		return body + sb.String()
	}
	return body[:idx] + sb.String()
}
