package dsl

import (
	"os"
	"regexp"
	"strings"

	"github.com/krewire/libs/markdown"
	"gopkg.in/yaml.v3"
)

var (
	styleRe    = regexp.MustCompile(`(?is)<style[^>]*>(.*?)</style>`)
	scriptRe   = regexp.MustCompile(`(?is)<script[^>]*>(.*?)</script>`)
	markdownRe = regexp.MustCompile(`(?is)<markdown[^>]*>(.*?)</markdown>`)
)

// KiwModule is the parsed result of a .kiw file.
// It is JSON-serializable and intentionally mirrors the JS parser output
// so the same .kiw file can be consumed from Go (html/template) and from
// JS/TS (string templates) without a custom toolchain.
type KiwModule struct {
	Frontmatter map[string]any `yaml:",inline" json:"frontmatter"`
	Body        string         `json:"body"`
	Styles      []string       `json:"styles"`
	Scripts     []string       `json:"scripts"`
	Markdown    []string       `json:"markdown"`
	Raw         string         `json:"-"`
}

// ParseKiw parses a .kiw file content into a KiwModule.
//
// Format (Astro-like, but YAML frontmatter for Go/JS native):
//
//	---
//	title: Landing
//	layout: Base
//	---
//	<h1>{{.Title}}</h1>
//	<style>h1{color:red}</style>
//	<script>console.log(1)</script>
//
// Frontmatter is optional YAML between leading ---\n ... ---\n.
// Body is html/template source with zero or more top-level <style> and <script>
// blocks extracted as scoped CSS / client JS.
func ParseKiw(src string) (*KiwModule, error) {
	m := &KiwModule{
		Frontmatter: map[string]any{},
		Raw:         src,
	}
	body := src

	if strings.HasPrefix(strings.TrimSpace(src), "---") {
		trimmed := strings.TrimLeft(src, "\r\n\t ")
		if strings.HasPrefix(trimmed, "---") {
			rest := trimmed[3:]
			// find closing ---\n
			idx := strings.Index(rest, "\n---")
			if idx >= 0 {
				fmRaw := rest[:idx]
				body = rest[idx+4:]
				// strip leading newline after closing ---
				body = strings.TrimLeft(body, "\r\n")
				var fm map[string]any
				if err := yaml.Unmarshal([]byte(fmRaw), &fm); err == nil && fm != nil {
					m.Frontmatter = fm
				}
			}
		}
	}

	styles := styleRe.FindAllStringSubmatch(body, -1)
	for _, sm := range styles {
		if len(sm) > 1 {
			m.Styles = append(m.Styles, strings.TrimSpace(sm[1]))
		}
	}
	body = styleRe.ReplaceAllString(body, "")

	scripts := scriptRe.FindAllStringSubmatch(body, -1)
	for _, sm := range scripts {
		if len(sm) > 1 {
			m.Scripts = append(m.Scripts, strings.TrimSpace(sm[1]))
		}
	}
	body = scriptRe.ReplaceAllString(body, "")

	markdowns := markdownRe.FindAllStringSubmatch(body, -1)
	for _, mm := range markdowns {
		if len(mm) > 1 {
			m.Markdown = append(m.Markdown, strings.TrimSpace(mm[1]))
		}
	}
	body = markdownRe.ReplaceAllStringFunc(body, func(match string) string {
		sub := markdownRe.FindStringSubmatch(match)
		if len(sub) < 2 {
			return ""
		}
		inner := strings.TrimSpace(sub[1])
		if inner == "" {
			return ""
		}
		rendered, _ := markdown.Render([]byte(inner))
		return "\n" + strings.TrimSpace(rendered) + "\n"
	})

	m.Body = strings.TrimSpace(body)
	return m, nil
}

// ParseKiwFile reads and parses a .kiw file at path.
func ParseKiwFile(path string) (*KiwModule, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseKiw(string(b))
}
