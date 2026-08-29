package test

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// HTMLAssert is an HTML-aware assertion helper (KWF-TST-U9K-010).
type HTMLAssert struct {
	t    *testing.T
	html string
	root *html.Node
}

var hashAssetRe = regexp.MustCompile(`assets/[^"']+\.[a-fA-F0-9]{6,12}\.(css|js)`)

// HTML parses html string and returns an assert helper.
func HTML(t *testing.T, htmlStr string) *HTMLAssert {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		t.Fatalf("HTML: parse failed %v", err)
	}
	return &HTMLAssert{t: t, html: htmlStr, root: doc}
}

// Has asserts that selector matches wantCount nodes.
func (h *HTMLAssert) Has(selector string, wantCount int) *HTMLAssert {
	h.t.Helper()
	got := h.Count(selector)
	if got != wantCount {
		h.t.Errorf("Has %q: got %d want %d", selector, got, wantCount)
	}
	return h
}

// Count returns the number of nodes matching selector.
func (h *HTMLAssert) Count(selector string) int {
	h.t.Helper()
	return len(findAll(h.root, selector))
}

// ContainsText asserts that html contains text (substring).
func (h *HTMLAssert) ContainsText(want string) *HTMLAssert {
	h.t.Helper()
	if !strings.Contains(h.html, want) {
		h.t.Errorf("ContainsText: %q not found in %q", want, h.html)
	}
	return h
}

// HasText asserts that the text content of selector contains want.
func (h *HTMLAssert) HasText(selector, want string) *HTMLAssert {
	h.t.Helper()
	nodes := findAll(h.root, selector)
	for _, n := range nodes {
		if strings.Contains(textContent(n), want) {
			return h
		}
	}
	h.t.Errorf("HasText %q: no node with text %q, nodes %d", selector, want, len(nodes))
	return h
}

// Attr asserts that selector's attr equals want.
func (h *HTMLAssert) Attr(selector, attr, want string) *HTMLAssert {
	h.t.Helper()
	nodes := findAll(h.root, selector)
	for _, n := range nodes {
		for _, a := range n.Attr {
			if a.Key == attr && a.Val == want {
				return h
			}
		}
	}
	h.t.Errorf("Attr %q %q: want %q not found", selector, attr, want)
	return h
}

// Snapshot performs a normalized golden comparison (KWF-TST-U9K-011).
func Snapshot(t *testing.T, name, htmlStr string) {
	t.Helper()
	normalized := normalizeHTML(htmlStr)
	Golden(t, name, normalized)
}

// GoldenHTML is an alias for Snapshot with .html.golden (KWF-TST-U9K-013).
func GoldenHTML(t *testing.T, name, htmlStr string) {
	t.Helper()
	Snapshot(t, name, htmlStr)
}

// ThemeSnapshot asserts Theme presence (KWF-TST-U9K-012).
func ThemeSnapshot(t *testing.T, name, htmlStr string) {
	t.Helper()
	if !strings.Contains(htmlStr, "data-theme") && !strings.Contains(htmlStr, "var(--primary)") && !strings.Contains(htmlStr, "--primary") {
		t.Errorf("ThemeSnapshot: html missing data-theme or --primary, got %q", htmlStr)
	}
	Snapshot(t, name, htmlStr)
}

func normalizeHTML(s string) string {
	// Strip hashed assets: assets/app.abc123.css -> assets/app.css
	s = hashAssetRe.ReplaceAllStringFunc(s, func(m string) string {
		// keep extension but strip hash
		if strings.HasSuffix(m, ".css") {
			return regexp.MustCompile(`\.[a-fA-F0-9]{6,12}\.css$`).ReplaceAllString(m, ".css")
		}
		if strings.HasSuffix(m, ".js") {
			return regexp.MustCompile(`\.[a-fA-F0-9]{6,12}\.js$`).ReplaceAllString(m, ".js")
		}
		return m
	})
	// Trim and collapse whitespace
	lines := strings.Split(s, "\n")
	var out []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	// Sort attrs inside tags is complex; for MVP just normalize whitespace and hash.
	// Also sort attributes alphabetically for stability: parse and re-render is heavy, skip for now.
	_ = sort.StringSlice{}
	return strings.Join(out, "\n")
}

func findAll(root *html.Node, selector string) []*html.Node {
	// Support descendant selector like "nav a" by splitting on space
	if strings.Contains(selector, " ") {
		parts := strings.Fields(selector)
		if len(parts) == 2 {
			ancestor, descendant := parts[0], parts[1]
			var res []*html.Node
			var walk func(*html.Node)
			walk = func(n *html.Node) {
				if n.Type == html.ElementNode && matchesSimple(n, descendant) && hasAncestor(n, ancestor) {
					res = append(res, n)
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
			}
			walk(root)
			return res
		}
	}
	var res []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && matches(n, selector) {
			res = append(res, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return res
}

func hasAncestor(n *html.Node, selector string) bool {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && matchesSimple(p, selector) {
			return true
		}
	}
	return false
}

func matchesSimple(n *html.Node, selector string) bool { return matches(n, selector) }

func matches(n *html.Node, selector string) bool {
	if selector == "" {
		return false
	}
	// #id
	if strings.HasPrefix(selector, "#") {
		id := selector[1:]
		for _, a := range n.Attr {
			if a.Key == "id" && a.Val == id {
				return true
			}
		}
		return false
	}
	// .class or tag.class
	tag := ""
	class := ""
	if strings.HasPrefix(selector, ".") {
		class = selector[1:]
	} else if strings.Contains(selector, ".") {
		parts := strings.SplitN(selector, ".", 2)
		tag = parts[0]
		class = parts[1]
	} else {
		tag = selector
	}
	if tag != "" && n.Data != tag {
		return false
	}
	if class != "" {
		for _, a := range n.Attr {
			if a.Key == "class" {
				classes := strings.Fields(a.Val)
				for _, c := range classes {
					if c == class {
						return true
					}
				}
				return false
			}
		}
		return false
	}
	return true
}

func textContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(b.String())
}
