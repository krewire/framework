package ssg

import (
	"fmt"
	"html/template"
	"reflect"

	"github.com/krewire/framework/ui"
)

// Config describes a static site declaratively. Decode it from ssg.yaml (or an
// equivalent document) and build it with BuildFromConfig, so projects ship a
// config file instead of a bespoke build command.
type Config struct {
	// Title is the site title, injected into every page's data as .Title.
	Title string `yaml:"title"`
	// Description is the site description, injected into every page's data as
	// .Description.
	Description string `yaml:"description"`
	// Version is the product/site version, injected into every page's data as
	// .Version so chrome and badges never hardcode it.
	Version string `yaml:"version"`
	// Output is the directory the site is exported to. Consumers may ignore it
	// and supply their own output path.
	Output string `yaml:"output"`
	// Theme configures the light/dark theme switcher and palette overrides.
	Theme *ThemeConfig `yaml:"theme"`
	// Components are the site's named components.
	Components []ComponentConfig `yaml:"components"`
	// Layouts are the site's named layouts.
	Layouts []LayoutConfig `yaml:"layouts"`
	// Pages are the site's output pages.
	Pages []PageConfig `yaml:"pages"`
	// Collections are content collections (Markdown + frontmatter).
	Collections []CollectionConfig `yaml:"collections"`
	// Assets maps output paths (e.g. "favicon.ico") to file contents.
	Assets map[string]string `yaml:"assets"`
	// Pipeline declares asset transforms (KWF-DR5YU): fingerprinting,
	// minification, image resize/format. Applied at build time.
	Pipeline []PipelineRule `yaml:"pipeline"`
	// Data is the site-wide data merged into every page's data value.
	Data map[string]any `yaml:"data"`
	// IncludeDrafts includes draft pages in build (dev only).
	IncludeDrafts bool `yaml:"include_drafts"`
	// IncludeFuture includes future-dated pages in build (dev only).
	IncludeFuture bool `yaml:"include_future"`
}

// ThemeConfig configures the theming system for a declarative site.
type ThemeConfig struct {
	// Default is the initial mode: auto, light, or dark.
	Default string `yaml:"default"`
	// Light and Dark map palette tokens (e.g. "primary") to color values,
	// overriding the ui defaults for each mode.
	Light map[string]string `yaml:"light"`
	Dark  map[string]string `yaml:"dark"`
}

// ComponentConfig is a component in a declarative Config.
type ComponentConfig struct {
	Name  string `yaml:"name"`
	Body  string `yaml:"body"`
	Style string `yaml:"style"`
}

// LayoutConfig is a layout in a declarative Config. Either Body/Style define
// a template layout, or UI references a reusable framework/ui shell.
type LayoutConfig struct {
	Name  string `yaml:"name"`
	Body  string `yaml:"body"`
	Style string `yaml:"style"`
	// UI, when set, builds the layout from a reusable ui.Layout shell instead
	// of Body/Style. The site theme is injected when the shell has none.
	UI *ui.LayoutConfig `yaml:"ui"`
}

// PageConfig is a page in a declarative Config.
type PageConfig struct {
	Path   string         `yaml:"path"`
	Title  string         `yaml:"title"`
	Layout string         `yaml:"layout"`
	Root   string         `yaml:"root"`
	Data   map[string]any `yaml:"data"`
}

// Site converts the config into a buildable Site. Page data merges the
// site-wide Data with page-specific overrides; Title, Description, and the
// configured Theme are injected so templates can reference them directly.
func (c *Config) Site() *Site {
	s := New().Funcs(template.FuncMap{
		// html marks a string as trusted HTML so templates can render data
		// fields that intentionally contain markup.
		"html": func(v any) template.HTML {
			return template.HTML(fmt.Sprint(v))
		},
		// markdown renders a Markdown string to trusted HTML.
		"markdown": markdownFunc,
	}).Registry(ui.Default())
	if c.Theme != nil {
		s.Asset("assets/theme.css", ui.ThemeModeVarsCSS+"\n"+ui.ThemeToggleCSS)
	}
	for _, comp := range c.Components {
		s.Component(Component{Name: comp.Name, Body: comp.Body, Style: comp.Style})
	}
	for _, l := range c.Layouts {
		if l.UI != nil {
			ul := ui.LayoutFromConfig(l.UI)
			if ul.Theme == nil && c.Theme != nil {
				ul.Theme = c.theme()
			}
			sl, err := LayoutFromUI(l.Name, *ul)
			if err != nil {
				s.Layout(Layout{Name: l.Name, Body: "{{if}}"})
				continue
			}
			s.Layout(sl)
			s.Asset("assets/ui.css", ui.ComponentsCSS())
			continue
		}
		s.Layout(Layout{Name: l.Name, Body: l.Body, Style: l.Style})
	}

	// Build collections
	collections := make(map[string]*Collection)
	collectionData := make(map[string]any)
	for _, cc := range c.Collections {
		col, err := BuildCollection(cc, ".")
		if err != nil {
			panic(fmt.Sprintf("collection %s: %v", cc.Name, err))
		}
		col.FilterDrafts(c.IncludeDrafts)
		col.FilterFuture(c.IncludeFuture)
		collections[cc.Name] = col
		// Build collection data for templates
		pagesData := make([]map[string]any, len(col.Pages))
		for i, p := range col.Pages {
			outputPath := col.GenerateOutputPath(p)
			pagesData[i] = map[string]any{
				"Title":       p.Config.Title,
				"Date":        p.RawDate,
				"Permalink":   outputPath,
				"Data":        p.Config.Extra,
				"Content":     p.Content,
				"Draft":       p.Config.Draft,
				"Tags":        p.Config.Tags,
				"Description": p.Config.Description,
			}
		}
		collectionData[cc.Name] = pagesData
	}

	base := c.pageData()
	if len(collectionData) > 0 {
		base["Collections"] = collectionData
	}

	for _, p := range c.Pages {
		s.Page(Page{Path: p.Path, Title: p.Title, Layout: p.Layout, Root: p.Root, Data: merge(base, p.Data)})
	}
	for name, body := range c.Assets {
		s.Asset(name, body)
	}
	if len(c.Pipeline) > 0 {
		s.Pipeline(c.Pipeline)
	}
	return s
}

// BuildFromConfig builds the declarative site into outDir and returns the
// paths created, relative to outDir.
func BuildFromConfig(cfg *Config, outDir string) ([]string, error) {
	return cfg.Site().Build(outDir)
}

// pageData builds the site-wide data injected into every page.
func (c *Config) pageData() map[string]any {
	data := map[string]any{}
	for k, v := range c.Data {
		data[k] = v
	}
	if c.Title != "" {
		data["Title"] = c.Title
	}
	if c.Description != "" {
		data["Description"] = c.Description
	}
	if c.Version != "" {
		data["Version"] = c.Version
	}
	if c.Theme != nil {
		data["Theme"] = c.theme()
	}
	data["IncludeDrafts"] = c.IncludeDrafts
	data["IncludeFuture"] = c.IncludeFuture
	return data
}

// theme converts the declarative theme into the ui theming value.
func (c *Config) theme() *ui.Theme {
	t := &ui.Theme{Default: c.Theme.Default}
	applyPalette(&t.Light, c.Theme.Light)
	applyPalette(&t.Dark, c.Theme.Dark)
	return t
}

// merge returns a copy of base overlaid with overrides.
func merge(base, overrides map[string]any) map[string]any {
	if len(overrides) == 0 {
		return base
	}
	out := make(map[string]any, len(base)+len(overrides))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}

// applyPalette applies the token overrides onto p using the ui palette css
// tags, so configs address tokens by their CSS name ("primary", "base-1",
// "error-content", …).
func applyPalette(p *ui.Palette, overrides map[string]string) {
	if p == nil || len(overrides) == 0 {
		return
	}
	kiw := reflect.ValueOf(p).Elem()
	rt := kiw.Type()
	for i := 0; i < kiw.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("css")
		if v, ok := overrides[tag]; ok && v != "" {
			kiw.Field(i).SetString(v)
		}
	}
}
