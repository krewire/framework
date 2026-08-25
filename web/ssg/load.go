package ssg

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/krewire/framework/dsl"
	"github.com/krewire/framework/ui"
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

	// Content collections: every dir under content/ is a collection
	collections := map[string]*Collection{}
	collectionData := map[string]any{}
	contentRoot := filepath.Join(root, "content")
	if st, err := os.Stat(contentRoot); err == nil && st.IsDir() {
		entries, _ := os.ReadDir(contentRoot)
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasPrefix(name, "_") {
				continue
			}
			cc := CollectionConfig{Name: name, Dir: filepath.Join("content", name)}
			col, err := BuildCollection(cc, root)
			if err != nil {
				continue
			}
			// Draft/future filtering: check meta or frontmatter? Use false for MVP (exclude drafts/future)
			col.FilterDrafts(false)
			col.FilterFuture(false)
			if len(col.Pages) == 0 {
				continue
			}
			collections[name] = col
			pagesData := make([]map[string]any, len(col.Pages))
			for i, p := range col.Pages {
				pagesData[i] = map[string]any{
					"Title":       p.Config.Title,
					"Date":        p.RawDate,
					"Permalink":   p.URL,
					"Data":        p.Config.Extra,
					"Content":     p.Content,
					"Draft":       p.Config.Draft,
					"Slug":        p.Config.Slug,
					"Description": p.Config.Description,
				}
			}
			collectionData[name] = pagesData
		}
	}

	// Inject collections into meta data for templates
	if len(collectionData) > 0 {
		if meta.Data == nil {
			meta.Data = map[string]any{}
		}
		meta.Data["Collections"] = collectionData
	}

	// Pages: handle [slug].kiw expansion
	for _, p := range findKiwFiles(filepath.Join(root, "pages")) {
		rel, _ := filepath.Rel(filepath.Join(root, "pages"), p)
		if strings.HasPrefix(filepath.Base(p), "_") || strings.Contains(rel, string(filepath.Separator)+"_") {
			continue
		}
		mod, err := dsl.ParseKiwFile(p)
		if err != nil {
			continue
		}
		// Check for [slug] dynamic route
		if strings.Contains(filepath.Base(p), "[slug]") {
			dir := filepath.Dir(rel)
			// dir is like blog or docs/blog
			collectionName := filepath.Base(dir)
			if dir == "." {
				collectionName = ""
			}
			// If dir is ".", then collection is from file name? Not needed for MVP
			col, ok := collections[collectionName]
			if !ok {
				continue
			}
			for _, item := range col.Pages {
				slug := item.Config.Slug
				if slug == "" {
					slug = strings.TrimSuffix(filepath.Base(item.Path), ".md")
				}
				route := "/" + filepath.ToSlash(filepath.Join(dir, slug))
				title := strFromFM(mod.Frontmatter, "title", item.Config.Title)
				if title == "" {
					title = meta.Title
				}
				layout := strFromFM(mod.Frontmatter, "layout", "")
				if layout == "" {
					// try layouts/<collection>.kiw, else Post, else Base
					if _, err := os.Stat(filepath.Join(root, "layouts", collectionName+".kiw")); err == nil {
						layout = collectionName
					} else if _, err := os.Stat(filepath.Join(root, "layouts", "Post.kiw")); err == nil {
						layout = "Post"
					} else if _, err := os.Stat(filepath.Join(root, "layouts", "Base.kiw")); err == nil {
						layout = "Base"
					}
				}
				data := mergeMeta(meta.Data, mod.Frontmatter)
				data["Title"] = title
				data["Content"] = item.Content
				data["Page"] = item
				for k, v := range item.Config.Extra {
					if _, ok := data[k]; !ok {
						data[k] = v
					}
				}
				body := mod.Body
				style := strings.Join(mod.Styles, "\n")
				compName := "page:" + route
				if style != "" {
					site.Component(Component{Name: compName, Body: body, Style: style})
					body = ""
				} else {
					compName = ""
					if body == "" {
						body = "<div></div>"
					}
				}
				rootName := compName
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
			}
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

	// Theme assets for file-based: emit theme.css if krewire.yaml has theme.
	// public/ files (loaded above) win over machine-generated assets at the
	// same path (KWF-DF3PL FRK-FLS-040).
	if meta.ThemeCSS != "" {
		if _, exists := site.AssetBody("assets/theme.css"); !exists {
			site.Asset("assets/theme.css", meta.ThemeCSS)
		}
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
		var raw struct {
			Title       string `yaml:"title"`
			Description string `yaml:"description"`
			Version     string `yaml:"version"`
			Theme       *struct {
				Default string            `yaml:"default"`
				Light   map[string]string `yaml:"light"`
				Dark    map[string]string `yaml:"dark"`
			} `yaml:"theme"`
		}
		if err := yaml.Unmarshal(b, &raw); err == nil {
			if raw.Title != "" {
				m.Title = raw.Title
			}
			if raw.Description != "" {
				m.Description = raw.Description
			}
			if raw.Version != "" {
				if m.Data == nil {
					m.Data = map[string]any{}
				}
				m.Data["Version"] = raw.Version
			}
			if raw.Theme != nil {
				t := &ui.Theme{Default: raw.Theme.Default}
				applyPalette(&t.Light, raw.Theme.Light)
				applyPalette(&t.Dark, raw.Theme.Dark)
				m.ThemeCSS = string(t.Style())
				if m.Data == nil {
					m.Data = map[string]any{}
				}
				m.Data["Theme"] = t
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
	ThemeCSS    string
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
	}
	return "/" + rel
}
