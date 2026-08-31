package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Env provides typed, <APP>_-prefixed environment configuration with explicit
// defaults and usage errors on invalid values (KWF-FGNZ9).
//
// Precedence is flags > environment > defaults. Callers resolve flag values
// first and only fall back to Env when a flag was not set, so the same key can
// be layered cleanly. Every getter coerces with strconv and never imports a
// non-stdlib configuration library.
type Env struct {
	prefix string
}

// NewEnv builds an Env for the named application. "greet" yields the prefix
// "GREET_"; non-alphanumeric characters collapse to underscores.
func NewEnv(appName string) *Env {
	return &Env{prefix: envPrefix(appName)}
}

// Prefix returns the resolved SCREAMING_SNAKE_CASE variable prefix, e.g. "GREET_".
func (e *Env) Prefix() string {
	return e.prefix
}

func envPrefix(name string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(name) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	s := strings.Trim(b.String(), "_")
	if s == "" {
		s = "APP"
	}
	return s + "_"
}

// GetString returns the variable value, or def when unset or empty.
func (e *Env) GetString(key, def string) string {
	if v, ok := os.LookupEnv(e.prefix + strings.ToUpper(key)); ok && v != "" {
		return v
	}
	return def
}

// GetBool returns the boolean variable or def. An invalid value returns an
// error the caller should surface as a usage error (exit code 2).
func (e *Env) GetBool(key string, def bool) (bool, error) {
	v, ok := os.LookupEnv(e.prefix + strings.ToUpper(key))
	if !ok || v == "" {
		return def, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def, fmt.Errorf("invalid %s%s: %q (want true/false)", e.prefix, strings.ToUpper(key), v)
	}
	return b, nil
}

// GetInt returns the integer variable or def. An invalid value returns an
// error the caller should surface as a usage error (exit code 2).
func (e *Env) GetInt(key string, def int) (int, error) {
	v, ok := os.LookupEnv(e.prefix + strings.ToUpper(key))
	if !ok || v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def, fmt.Errorf("invalid %s%s: %q (want an integer)", e.prefix, strings.ToUpper(key), v)
	}
	return n, nil
}
