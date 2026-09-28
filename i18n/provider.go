package i18n

import (
	"github.com/krewire/framework/app"
)

// Provider returns an app.Provider that registers *Bundle and *Translator
// singletons into the application container. If a Config or *Config is registered
// in the container, it is applied to the bundle automatically.
func Provider(bundle *Bundle) app.Provider {
	return app.ProviderFunc(func(c *app.Container) error {
		if err := app.Singleton[*Bundle](c, func() *Bundle {
			// Auto-apply Config if present in container
			if cfg, err := app.Resolve[Config](c); err == nil {
				WithConfig(cfg)(bundle)
			} else if cfgPtr, err := app.Resolve[*Config](c); err == nil && cfgPtr != nil {
				WithConfig(*cfgPtr)(bundle)
			}
			return bundle
		}); err != nil {
			return err
		}

		return app.Singleton[*Translator](c, func() *Translator {
			b, err := app.Resolve[*Bundle](c)
			if err != nil || b == nil {
				b = bundle
			}
			if b == nil {
				return NewTranslator(nil, "en")
			}
			return b.ForLocale(b.DefaultLocale())
		})
	})
}

// ProviderWithConfig returns an app.Provider that explicitly binds an i18n Config,
// *Bundle, and *Translator into the application container.
func ProviderWithConfig(bundle *Bundle, cfg Config) app.Provider {
	return app.ProviderFunc(func(c *app.Container) error {
		if err := app.Singleton[Config](c, func() Config {
			return cfg
		}); err != nil {
			return err
		}
		if bundle != nil {
			WithConfig(cfg)(bundle)
		}
		return Provider(bundle).Register(c)
	})
}
