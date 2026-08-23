// Package config defines the typed application configuration
// (FRK-STR-005): a struct loaded through libs/config and validated with
// libs/validate. The config file is krewire.yaml at the project root.
package config

import (
	"fmt"
	"os"

	rconfig "github.com/krewire/libs/config"
	"github.com/krewire/libs/validate"
)

// Config is the typed application configuration.
type Config struct {
	Addr  string `yaml:"addr" env:"ADDR" validate:"required"`
	Title string `yaml:"title" env:"TITLE" validate:"required"`
}

// Load reads krewire.yaml from path, overlays the environment, and returns a
// validated Config.
func Load(path string) (*Config, error) {
	cfg := &Config{Addr: ":8080", Title: "Krewire Monolith"}
	if err := rconfig.Load(path, cfg); err != nil {
		return nil, err
	}
	if err := rconfig.Override(cfg, os.LookupEnv); err != nil {
		return nil, err
	}
	if err := validate.Struct(cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}
