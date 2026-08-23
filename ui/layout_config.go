package ui

import "html/template"

// LayoutConfig is the declarative form of a Layout. Decode it from YAML or
// JSON and convert with LayoutFromConfig. Parsing itself happens in the
// consumer (e.g. the krewire CLI), mirroring how web/ssg.Config works.
type LayoutConfig struct {
	Title       string        `yaml:"title"`
	Description string        `yaml:"description"`
	Head        template.HTML `yaml:"head"` // custom <head> content (fonts, meta)
	BodyAttrs   template.HTML `yaml:"body_attrs"`
	Stylesheets []string      `yaml:"stylesheets"` // assets/ui.css, assets/style.css, …
	Header      *HeaderConfig `yaml:"header"`
	Main        ComponentRef  `yaml:"main"`
	Footer      *FooterConfig `yaml:"footer"`
	Aside       ComponentRef  `yaml:"aside"`
	Theme       *ThemeConfig  `yaml:"theme"`
}

// ComponentRef references a named component or embeds raw HTML.
type ComponentRef struct {
	// Component is the name of a component registered in the enclosing
	// declarative config.
	Component string `yaml:"component"`
	// HTML is a literal pre-rendered fragment used when Component is empty.
	HTML template.HTML `yaml:"html"`
}

// NavLinkConfig is a declarative navigation entry.
type NavLinkConfig struct {
	Text   string `yaml:"text"`
	URL    string `yaml:"url"`
	Active bool   `yaml:"active"`
	Target string `yaml:"target"`
	Rel    string `yaml:"rel"`
	Class  string `yaml:"class"`
}

// NavConfig is the declarative form of Nav.
type NavConfig struct {
	Label string          `yaml:"label"`
	Links []NavLinkConfig `yaml:"links"`
}

// HeaderConfig is the declarative form of Header.
type HeaderConfig struct {
	// Brand is the brand markup rendered in the header's left slot.
	Brand template.HTML `yaml:"brand"`
	// Nav configures the navigation links.
	Nav *NavConfig `yaml:"nav"`
	// Actions is raw markup for the header's right slot. When ThemeToggle is
	// true, the layout's theme button is appended after it.
	Actions template.HTML `yaml:"actions"`
	// ThemeToggle toggles whether the theme switcher renders in the header.
	ThemeToggle bool `yaml:"theme_toggle"`
}

// FooterConfig is the declarative form of Footer.
type FooterConfig struct {
	Left   template.HTML `yaml:"left"`
	Center template.HTML `yaml:"center"`
	Right  template.HTML `yaml:"right"`
}

// LayoutFromConfig converts a declarative layout config into a Layout.
func LayoutFromConfig(cfg *LayoutConfig) *Layout {
	l := &Layout{
		Title:       cfg.Title,
		Description: cfg.Description,
		Head:        cfg.Head,
		BodyAttrs:   cfg.BodyAttrs,
		Stylesheets: cfg.Stylesheets,
	}
	if cfg.Theme != nil {
		l.Theme = cfg.Theme.Theme()
	}
	if cfg.Header != nil {
		l.Header = headerFromConfig(cfg.Header, l.Theme)
	}
	if cfg.Main.HTML != "" {
		l.Main = cfg.Main.HTML
	}
	if cfg.Footer != nil {
		f := Footer{
			Left:   cfg.Footer.Left,
			Center: cfg.Footer.Center,
			Right:  cfg.Footer.Right,
		}
		l.Footer = f.Render()
	}
	if cfg.Aside.HTML != "" {
		l.Aside = cfg.Aside.HTML
	}
	return l
}

// headerFromConfig builds Header markup from its declarative form, appending
// the layout's theme button when ThemeToggle is set.
func headerFromConfig(cfg *HeaderConfig, theme *Theme) template.HTML {
	h := Header{Brand: cfg.Brand}
	if cfg.Nav != nil {
		links := make([]NavLink, 0, len(cfg.Nav.Links))
		for _, l := range cfg.Nav.Links {
			links = append(links, NavLink{Text: l.Text, URL: l.URL, Active: l.Active, Target: l.Target, Rel: l.Rel, Class: l.Class})
		}
		h.Nav = Nav{Links: links, Label: cfg.Nav.Label}.Render()
	}
	if cfg.Actions != "" {
		h.Actions = cfg.Actions
	}
	if cfg.ThemeToggle {
		if theme == nil {
			theme = &Theme{}
		}
		h.Actions = Slot(h.Actions, theme.Button())
	}
	return h.Render()
}

// ThemeConfig is the declarative form of ui.Theme.
type ThemeConfig struct {
	Default string            `yaml:"default"`
	Light   map[string]string `yaml:"light"`
	Dark    map[string]string `yaml:"dark"`
}

// Theme converts the declarative theme into a ui.Theme using the shipped
// default palettes as the base for unset tokens.
func (c *ThemeConfig) Theme() *Theme {
	t := &Theme{Default: c.Default}
	t.Light = paletteFrom(c.Light, DefaultLightPalette)
	t.Dark = paletteFrom(c.Dark, DefaultDarkPalette)
	return t
}

// paletteFrom fills a Palette from token overrides, starting from defaults.
func paletteFrom(overrides map[string]string, def Palette) Palette {
	p := def
	if len(overrides) == 0 {
		return p
	}
	for token, v := range overrides {
		if v == "" {
			continue
		}
		switch token {
		case "base-1":
			p.Base1 = Color(v)
		case "base-1-content":
			p.Base1Content = Color(v)
		case "base-2":
			p.Base2 = Color(v)
		case "base-2-content":
			p.Base2Content = Color(v)
		case "base-3":
			p.Base3 = Color(v)
		case "base-3-content":
			p.Base3Content = Color(v)
		case "primary":
			p.Primary = Color(v)
		case "primary-content":
			p.PrimaryContent = Color(v)
		case "secondary":
			p.Secondary = Color(v)
		case "secondary-content":
			p.SecondaryContent = Color(v)
		case "accent":
			p.Accent = Color(v)
		case "accent-content":
			p.AccentContent = Color(v)
		case "ghost":
			p.Ghost = Color(v)
		case "ghost-content":
			p.GhostContent = Color(v)
		case "neutral":
			p.Neutral = Color(v)
		case "neutral-content":
			p.NeutralContent = Color(v)
		case "success":
			p.Success = Color(v)
		case "success-content":
			p.SuccessContent = Color(v)
		case "info":
			p.Info = Color(v)
		case "info-content":
			p.InfoContent = Color(v)
		case "warning":
			p.Warning = Color(v)
		case "warning-content":
			p.WarningContent = Color(v)
		case "error":
			p.Error = Color(v)
		case "error-content":
			p.ErrorContent = Color(v)
		}
	}
	return p
}
