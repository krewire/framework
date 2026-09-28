package i18n

import (
	"github.com/krewire/framework/app"
)

// Provider returns an app.Provider that registers *Bundle and *Translator
// singletons into the application container.
func Provider(bundle *Bundle) app.Provider {
	return app.ProviderFunc(func(c *app.Container) error {
		if err := app.Singleton[*Bundle](c, func() *Bundle {
			return bundle
		}); err != nil {
			return err
		}

		return app.Singleton[*Translator](c, func() *Translator {
			if bundle == nil {
				return NewTranslator(nil, "en")
			}
			return bundle.ForLocale(bundle.DefaultLocale())
		})
	})
}
