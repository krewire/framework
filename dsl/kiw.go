package dsl

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/krewire/libs/markdown"
	"gopkg.in/yaml.v3"
)

var (
	styleRe           = regexp.MustCompile(`(?is)<style([^>]*)>(.*?)</style>`)
	scriptRe          = regexp.MustCompile(`(?is)<script([^>]*)>(.*?)</script>`)
	markdownRe        = regexp.MustCompile(`(?is)<markdown[^>]*>(.*?)</markdown>`)
	attrRe            = regexp.MustCompile("([a-zA-Z_:][-a-zA-Z0-9_:.]*)(?:\\s*=\\s*(?:\"([^\"]*)\"|'([^']*)'|([^\\s\"'=<>`]+)))?")
	selfClosingCompRe = regexp.MustCompile(`(?s)<([A-Z][a-zA-Z0-9_]*)([^>]*?)\s*/>`)
	openCompTagRe     = regexp.MustCompile(`<([A-Z][a-zA-Z0-9_]*)([^>]*)>`)
	mustacheCompRe    = regexp.MustCompile(`{{\s*component\s+"([^"]+)"(?:\s+([^{}]+))?\s*}}`)
	compAttrRe        = regexp.MustCompile("([a-zA-Z_:][-a-zA-Z0-9_:.]*)(?:\\s*=\\s*(?:\"([^\"]*)\"|'([^']*)'|\\{([^}]*)\\}|([^\\s\"'=<>`]+)))?")
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

// ComponentCall records a parsed component invocation in a .kiw template,
// whether written as {{component "Name" ...}} or <Name ... />.
type ComponentCall struct {
	Name  string            `json:"name"`
	Props map[string]string `json:"props,omitempty"`
	Raw   string            `json:"raw"`
}

// KiwModule is the parsed result of a .kiw file.
// It is JSON-serializable and intentionally mirrors the JS parser output
// so the same .kiw file can be consumed from Go (html/template) and from
// JS/TS (string templates) without a custom toolchain.
type KiwModule struct {
	Frontmatter    map[string]any  `yaml:",inline" json:"frontmatter"`
	Body           string          `json:"body"`
	Styles         []string        `json:"styles"`
	Scripts        []string        `json:"scripts"`
	StyleBlocks    []StyleBlock    `json:"styleBlocks"`
	ScriptBlocks   []ScriptBlock   `json:"scriptBlocks"`
	Markdown       []string        `json:"markdown"`
	Components     []ComponentCall `json:"components,omitempty"`
	ComponentNames []string        `json:"componentNames,omitempty"`
	Raw            string          `json:"-"`
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

func isNumeric(s string) bool {
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return false
}

func desugarComponent(name, attrStr, bodyContent, raw string) (string, ComponentCall) {
	call := ComponentCall{
		Name:  name,
		Props: make(map[string]string),
		Raw:   raw,
	}

	trimmedAttr := strings.TrimSpace(attrStr)
	trimmedBody := strings.TrimSpace(bodyContent)

	if trimmedAttr == "" && trimmedBody == "" {
		return fmt.Sprintf(`{{component %q}}`, name), call
	}

	if trimmedAttr == "." && trimmedBody == "" {
		call.Props["."] = "."
		return fmt.Sprintf(`{{component %q .}}`, name), call
	}

	if strings.HasPrefix(trimmedAttr, "(dict") && trimmedBody == "" {
		return fmt.Sprintf(`{{component %q %s}}`, name, trimmedAttr), call
	}

	var dictPairs []string
	if trimmedAttr != "" {
		matches := compAttrRe.FindAllStringSubmatch(attrStr, -1)
		for _, sm := range matches {
			if len(sm) < 2 || strings.TrimSpace(sm[1]) == "" {
				continue
			}
			key := sm[1]
			// boolean flag without explicit value (e.g. <Navbar ShowSidebarToggle />)
			if (len(sm) < 3 || (sm[2] == "" && sm[3] == "" && sm[4] == "" && sm[5] == "")) && !strings.Contains(sm[0], "=") {
				call.Props[key] = "true"
				dictPairs = append(dictPairs, fmt.Sprintf("%q true", key))
				continue
			}

			val := ""
			isExpr := false
			if len(sm) > 2 && sm[2] != "" {
				val = sm[2]
				if strings.HasPrefix(val, "{{") && strings.HasSuffix(val, "}}") {
					val = strings.TrimSpace(val[2 : len(val)-2])
					isExpr = true
				} else if strings.HasPrefix(val, "{") && strings.HasSuffix(val, "}") {
					val = strings.TrimSpace(val[1 : len(val)-1])
					isExpr = true
				}
			} else if len(sm) > 3 && sm[3] != "" {
				val = sm[3]
			} else if len(sm) > 4 && sm[4] != "" {
				val = strings.TrimSpace(sm[4])
				isExpr = true
			} else if len(sm) > 5 && sm[5] != "" {
				val = sm[5]
				if val == "true" || val == "false" || isNumeric(val) || strings.HasPrefix(val, ".") || strings.HasPrefix(val, "$") {
					isExpr = true
				}
			}

			call.Props[key] = val
			if isExpr {
				dictPairs = append(dictPairs, fmt.Sprintf("%q %s", key, val))
			} else if val == "true" || val == "false" || isNumeric(val) {
				dictPairs = append(dictPairs, fmt.Sprintf("%q %s", key, val))
			} else {
				dictPairs = append(dictPairs, fmt.Sprintf("%q %q", key, val))
			}
		}
	}

	if trimmedBody != "" {
		call.Props["Body"] = trimmedBody
		dictPairs = append(dictPairs, fmt.Sprintf("%q %q", "Body", trimmedBody))
	}

	if len(dictPairs) == 0 {
		return fmt.Sprintf(`{{component %q}}`, name), call
	}
	return fmt.Sprintf(`{{component %q (dict %s)}}`, name, strings.Join(dictPairs, " ")), call
}

// DesugarTemplate translates JSX-like component tags (<ComponentName ... />)
// into Go template component invocations ({{component "ComponentName" ...}}).
func DesugarTemplate(src string) (string, error) {
	// First desugar self-closing components: <Navbar ... />
	res := selfClosingCompRe.ReplaceAllStringFunc(src, func(match string) string {
		sm := selfClosingCompRe.FindStringSubmatch(match)
		if len(sm) < 3 {
			return match
		}
		name := sm[1]
		attrStr := sm[2]
		replacement, _ := desugarComponent(name, attrStr, "", match)
		return replacement
	})

	// Then desugar paired components: <Card ...>...</Card>
	res = desugarPairedComponents(res)

	return res, nil
}

func desugarPairedComponents(src string) string {
	res := src
	for {
		m := openCompTagRe.FindStringSubmatchIndex(res)
		if m == nil {
			break
		}
		tagName := res[m[2]:m[3]]
		attrStr := res[m[4]:m[5]]

		// Check if it's actually self-closing with slash at end of attrs: <Tag .../>
		trimmedAttr := strings.TrimSpace(attrStr)
		if strings.HasSuffix(trimmedAttr, "/") {
			realAttr := strings.TrimSuffix(trimmedAttr, "/")
			fullTag := res[m[0]:m[1]]
			replacement, _ := desugarComponent(tagName, realAttr, "", fullTag)
			res = res[:m[0]] + replacement + res[m[1]:]
			continue
		}

		closeTag := "</" + tagName + ">"
		closeIdx := strings.Index(res[m[1]:], closeTag)
		if closeIdx == -1 {
			// No matching closing tag, treat as self-closing
			fullTag := res[m[0]:m[1]]
			replacement, _ := desugarComponent(tagName, attrStr, "", fullTag)
			res = res[:m[0]] + replacement + res[m[1]:]
			continue
		}
		closeIdx += m[1]
		bodyContent := res[m[1]:closeIdx]
		fullTag := res[m[0] : closeIdx+len(closeTag)]
		replacement, _ := desugarComponent(tagName, attrStr, bodyContent, fullTag)
		res = res[:m[0]] + replacement + res[closeIdx+len(closeTag):]
	}
	return res
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
//	<Navbar />
//	<style>h1{color:red}</style>
//	<script>console.log(1)</script>
//
// Frontmatter is optional YAML between leading ---\n ... ---\n.
// Body is html/template source with zero or more top-level <style> and <script>
// blocks extracted as scoped CSS / client JS.
// Both {{component "Name"}} and <ComponentName /> are recognized and desugared
// to uniform component calls.
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

	// Desugar JSX-like component tags (<ComponentName ... />) to {{component "ComponentName" ...}}
	desugared, _ := DesugarTemplate(body)
	body = desugared

	// Extract all component invocations (both original {{component ...}} and desugared <Tag />)
	nameSet := make(map[string]bool)
	for _, match := range mustacheCompRe.FindAllStringSubmatch(body, -1) {
		if len(match) < 2 {
			continue
		}
		compName := match[1]
		m.Components = append(m.Components, ComponentCall{
			Name: compName,
			Raw:  match[0],
		})
		nameSet[compName] = true
	}

	for name := range nameSet {
		m.ComponentNames = append(m.ComponentNames, name)
	}
	sort.Strings(m.ComponentNames)

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
