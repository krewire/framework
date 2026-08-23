package dsl

import (
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	styleRe  = regexp.MustCompile(`(?is)<style[^>]*>(.*?)</style>`)
	scriptRe = regexp.MustCompile(`(?is)<script[^>]*>(.*?)</script>`)
)

// KiwModule is the parsed result of a .kiw file.
// It is JSON-serializable and intentionally mirrors the JS parser output
// so the same .kiw file can be consumed from Go (html/template) and from
// JS/TS (string templates) without a custom toolchain.
type KiwModule struct {
	Frontmatter map[string]any `yaml:",inline" json:"frontmatter"`
	Body        string          `json:"body"`
	Styles      []string        `json:"styles"`
	Scripts     []string        `json:"scripts"`
	Raw         string          `json:"-"`
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
