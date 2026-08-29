package dsl

import (
	"os"
	"regexp"
	"strings"

	"github.com/krewire/libs/markdown"
	"gopkg.in/yaml.v3"
)

var (
	styleRe    = regexp.MustCompile(`(?is)<style([^>]*)>(.*?)</style>`)
	scriptRe   = regexp.MustCompile(`(?is)<script([^>]*)>(.*?)</script>`)
	markdownRe = regexp.MustCompile(`(?is)<markdown[^>]*>(.*?)</markdown>`)
	attrRe     = regexp.MustCompile("([a-zA-Z_:][-a-zA-Z0-9_:.]*)(?:\\s*=\\s*(?:\"([^\"]*)\"|'([^']*)'|([^\\s\"'=<>`]+)))?")
)

// StyleBlock holds a scoped style block with its attributes.
type StyleBlock struct {
	Lang    string `json:"lang"`
	Scoped  bool   `json:"scoped"`
	Content string `json:"content"`
}

// ScriptBlock holds a script block with tier attributes (FRK-DSL-030/031).
// Lang is js|ts|go|rust, Hydrate is load|idle|visible, Server/Compute flags.
type ScriptBlock struct {
	Lang    string `json:"lang"`
	Hydrate string `json:"hydrate"`
	Server  bool   `json:"server"`
	Compute bool   `json:"compute"`
	Content string `json:"content"`
}

// KiwModule is the parsed result of a .kiw file.
// It is JSON-serializable and intentionally mirrors the JS parser output
// so the same .kiw file can be consumed from Go (html/template) and from
// JS/TS (string templates) without a custom toolchain.
type KiwModule struct {
	Frontmatter  map[string]any `yaml:",inline" json:"frontmatter"`
	Body         string         `json:"body"`
	Styles       []string       `json:"styles"`
	Scripts      []string       `json:"scripts"`
	StyleBlocks  []StyleBlock   `json:"styleBlocks"`
	ScriptBlocks []ScriptBlock  `json:"scriptBlocks"`
	Markdown     []string       `json:"markdown"`
	Raw          string         `json:"-"`
}

func parseAttrs(tag string) map[string]string {
	m := map[string]string{}
	for _, sm := range attrRe.FindAllStringSubmatch(tag, -1) {
		if len(sm) < 2 {
			continue
		}
		key := strings.ToLower(sm[1])
		val := ""
		if len(sm) > 2 && sm[2] != "" {
			val = sm[2]
		} else if len(sm) > 3 && sm[3] != "" {
			val = sm[3]
		} else if len(sm) > 4 && sm[4] != "" {
			val = sm[4]
		}
		m[key] = val
	}
	return m
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
		if len(sm) < 3 {
			continue
		}
		attrs := parseAttrs(sm[1])
		content := strings.TrimSpace(sm[2])
		m.Styles = append(m.Styles, content)
		lang := attrs["lang"]
		if lang == "" {
			lang = "css"
		}
		m.StyleBlocks = append(m.StyleBlocks, StyleBlock{
			Lang:    strings.ToLower(lang),
			Scoped:  false,
			Content: content,
		})
		if _, ok := attrs["scoped"]; ok {
			m.StyleBlocks[len(m.StyleBlocks)-1].Scoped = true
		}
	}
	body = styleRe.ReplaceAllString(body, "")

	scripts := scriptRe.FindAllStringSubmatch(body, -1)
	for _, sm := range scripts {
		if len(sm) < 3 {
			continue
		}
		attrs := parseAttrs(sm[1])
		content := strings.TrimSpace(sm[2])
		m.Scripts = append(m.Scripts, content)
		lang := strings.ToLower(attrs["lang"])
		hydrate := strings.ToLower(attrs["hydrate"])
		_, hasServer := attrs["server"]
		_, hasCompute := attrs["compute"]
		if lang == "" {
			lang = "js"
		}
		if lang == "js" || lang == "ts" || lang == "go" {
			if !hasServer && !hasCompute && hydrate == "" {
				hydrate = "load"
			}
		}
		m.ScriptBlocks = append(m.ScriptBlocks, ScriptBlock{
			Lang:    lang,
			Hydrate: hydrate,
			Server:  hasServer,
			Compute: hasCompute,
			Content: content,
		})
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
