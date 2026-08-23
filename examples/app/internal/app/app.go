// Package app is the assembly root (FRK-STR-004): it composes every
// dependency through the app container (KWF-D1CNT) and returns the ready
// *web.App. cmd only calls this and runs.
package app

import (
	"io/fs"

	rvapp "github.com/krewire/framework/app"
	"github.com/krewire/framework/examples/app"
	"github.com/krewire/framework/examples/app/internal/config"
	"github.com/krewire/framework/examples/app/internal/http"
	"github.com/krewire/framework/examples/app/web/layouts"
	"github.com/krewire/framework/examples/app/web/pages"
	"github.com/krewire/framework/examples/app/web/theme"
	rvweb "github.com/krewire/framework/web"
)

// webProvider assembles the SSR side of the app: theme, layout, pages, and
// embedded static assets.
type webProvider struct{}

// Register binds the web app singleton.
func (p *webProvider) Register(c *rvapp.Container) error {
	return rvapp.Singleton[*rvweb.App](c, func() *rvweb.App { return rvweb.NewApp() })
}

// Boot configures the app once the container can resolve it.
func (p *webProvider) Boot(c *rvapp.Container) error {
	a, err := rvapp.Resolve[*rvweb.App](c)
	if err != nil {
		return err
	}
	a.Theme(theme.New())
	a.Layout(layouts.Shell())

	sub, err := fs.Sub(public.Files, "public")
	if err != nil {
		return err
	}
	a.Static("/static", sub)
	pages.Register(a)
	return nil
}

// New assembles the full application and returns the ready *web.App
// (KWF-D1CNT FRK-CNT-012, FRK-STR-004).
func New(cfg *config.Config) (*rvweb.App, error) {
	container, err := rvapp.NewApp(
		&configProvider{cfg: cfg},
		&webProvider{},
		&http.Provider{},
	).Build()
	if err != nil {
		return nil, err
	}
	return rvapp.Resolve[*rvweb.App](container)
}

type configProvider struct {
	cfg *config.Config
}

// Register binds the loaded config into the container.
func (p *configProvider) Register(c *rvapp.Container) error {
	return rvapp.Singleton[*config.Config](c, func() *config.Config { return p.cfg })
}

// Boot does nothing for config.
func (p *configProvider) Boot(c *rvapp.Container) error { return nil }
