package storage

import (
	"github.com/krewire/framework/app"
)

// Provider returns an app.Provider registering kv as the container-wide KV
// singleton so application modules resolve storage.KV during assembly.
func Provider(kv KV) app.Provider {
	return app.ProviderFunc(func(c *app.Container) error {
		return app.Singleton[*KV](c, func() *KV { return &kv })
	})
}
