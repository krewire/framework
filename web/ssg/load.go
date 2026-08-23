package ssg

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/krewire/framework/dsl"
	"gopkg.in/yaml.v3"
)

func LoadFromDir(root string) (*Site, error) {
	site := New()
	meta := loadMeta(root)

	for _, p := range findKiwFiles(filepath.Join(root, "components")) {
		mod, err := dsl.ParseKiwFile(p)
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(p), ".kiw")
		if strings.HasPrefix(name, "_") {
			continue
		}
		style := strings.Join(mod.Styles, "\n")
		site.Component(Component{Name: name, Body: mod.Body, Style: style})
	}

	for _, p := range findKiwFiles(filepath.Join(root, "layouts")) {
		mod, err := dsl.ParseKiwFile(p)
		if err != nil {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(p), ".kiw")
		if strings.HasPrefix(name, "_") {
			continue
		}
		style := strings.Join(mod.Styles, "\n")
		site.Layout(Layout{Name: name, Body: mod.Body, Style: style})
	}

	for _, p := range findKiwFiles(filepath.Join(root, "pages")) {
		rel, _ := filepath.Rel(filepath.Join(root, "pages"), p)
		if strings.HasPrefix(filepath.Base(p), "_") || strings.Contains(rel, string(filepath.Separator)+"_") {
			continue
		}
		mod, err := dsl.ParseKiwFile(p)
		if err != nil {
			continue
		}
		route := routeFromRel(rel)
		title := strFromFM(mod.Frontmatter, "title", meta.Title)
		layout := strFromFM(mod.Frontmatter, "layout", "")
		if layout == "" {
			if _, err := os.Stat(filepath.Join(root, "layouts", "Base.kiw")); err == nil {
				layout = "Base"
			}
		}
		data := mergeMeta(meta.Data, mod.Frontmatter)
		data["Title"] = title
		if d := strFromFM(mod.Frontmatter, "description", ""); d != "" {
			data["Description"] = d
		}
		body := mod.Body
		style := strings.Join(mod.Styles, "\n")
		pageCompName := "page:" + route
		if style != "" {
			site.Component(Component{Name: pageCompName, Body: body, Style: style})
			body = ""
		} else {
			pageCompName = ""
			if body == "" {
				body = "<div></div>"
			}
		}
		rootName := pageCompName
		if rootName == "" {
			rootName = "page:" + route + ":body"
			site.Component(Component{Name: rootName, Body: body})
		}
		site.Page(Page{
			Path:   route,
			Title:  title,
			Layout: layout,
			Root:   rootName,
			Data:   data,
		})
		for i, sc := range mod.Scripts {
			site.Asset(filepath.Join("assets", "page"+strings.ReplaceAll(route, "/", "-")+string(rune('0'+i))+".js"), sc)
		}
	}

	publicDir := filepath.Join(root, "public")
	if st, err := os.Stat(publicDir); err == nil && st.IsDir() {
		_ = filepath.WalkDir(publicDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(publicDir, path)
			b, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			site.Asset(rel, string(b))
			return nil
		})
	}

	return site, nil
}

func loadMeta(root string) *meta {
	m := &meta{Data: map[string]any{}}
	for _, name := range []string{"krewire.yaml", "krewire.yml"} {
		b, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		var raw map[string]any
		if err := yaml.Unmarshal(b, &raw); err == nil {
			if v, ok := raw["title"].(string); ok {
				m.Title = v
			}
			if v, ok := raw["description"].(string); ok {
				m.Description = v
			}
		}
		break
	}
	return m
}

type meta struct {
	Title       string
	Description string
	Data        map[string]any
}

func mergeMeta(base map[string]any, fm map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range fm {
		out[k] = v
	}
	return out
}

func strFromFM(fm map[string]any, key, fallback string) string {
	if v, ok := fm[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return strings.Trim(s, `"'`)
		}
	}
	return fallback
}

func findKiwFiles(dir string) []string {
	var out []string
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), ".kiw") {
			out = append(out, path)
		}
		return nil
	})
	return out
}

func routeFromRel(rel string) string {
	rel = filepath.ToSlash(rel)
	rel = strings.TrimSuffix(rel, ".kiw")
	if rel == "index" {
		return "/"
	}
	if strings.HasSuffix(rel, "/index") {
		rel = strings.TrimSuffix(rel, "/index")
		return "/" + rel + "/"
	}
	return "/" + rel + "/"
}
