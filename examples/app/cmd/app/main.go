// Example fullstack monolith application: one binary serves SSR pages, JSON
// APIs, and static assets through the same web.App runtime, assembled through
// the app container (KWF-D1CNT) and laid out canonically (KWF-M1XKZ).
//
// Run with:
//
//	go run ./examples/app/cmd/app
//
// Pages render through the same pipeline as the static exporter, so a static
// page registered below could also be exported (see web.App.Export)
// byte-identically.
package main

import (
	"fmt"
	"os"

	"github.com/krewire/framework/examples/app/internal/app"
	"github.com/krewire/framework/examples/app/internal/config"
)

func main() {
	cfg, err := config.Load("krewire.yaml")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	a, err := app.New(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := a.Run(cfg.Addr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
