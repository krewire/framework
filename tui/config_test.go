package tui

import (
	"testing"
)

func TestEnv_PrefixSanitizesName(t *testing.T) {
	cases := map[string]string{
		"greet":   "GREET_",
		"my-app":  "MY_APP_",
		"kiw.dev": "KIW_DEV_",
	}
	for name, want := range cases {
		if got := NewEnv(name).Prefix(); got != want {
			t.Errorf("NewEnv(%q).Prefix() = %q, want %q", name, got, want)
		}
	}
}

func TestEnv_GetString_DefaultAndOverride(t *testing.T) {
	t.Setenv("GREET_GREETING", "Halo")
	if got := NewEnv("greet").GetString("greeting", "Hello"); got != "Halo" {
		t.Errorf("GetString = %q, want %q", got, "Halo")
	}
	t.Setenv("GREET_GREETING", "")
	if got := NewEnv("greet").GetString("greeting", "Hello"); got != "Hello" {
		t.Errorf("GetString empty falls back to default = %q, want %q", got, "Hello")
	}
	if got := NewEnv("greet").GetString("missing", "dflt"); got != "dflt" {
		t.Errorf("GetString missing = %q, want %q", got, "dflt")
	}
}

func TestEnv_GetBool(t *testing.T) {
	e := NewEnv("greet")
	if v, _ := e.GetBool("flag", true); !v {
		t.Error("GetBool default true")
	}
	t.Setenv("GREET_FLAG", "false")
	if v, _ := e.GetBool("flag", true); v {
		t.Error("GetBool override false")
	}
	t.Setenv("GREET_FLAG", "nope")
	if _, err := e.GetBool("flag", true); err == nil {
		t.Error("GetBool invalid must error")
	}
}

func TestEnv_GetInt(t *testing.T) {
	e := NewEnv("greet")
	if v, _ := e.GetInt("port", 8080); v != 8080 {
		t.Errorf("GetInt default = %d, want 8080", v)
	}
	t.Setenv("GREET_PORT", "3000")
	if v, _ := e.GetInt("port", 8080); v != 3000 {
		t.Errorf("GetInt override = %d, want 3000", v)
	}
	t.Setenv("GREET_PORT", "notanint")
	if _, err := e.GetInt("port", 8080); err == nil {
		t.Error("GetInt invalid must error")
	}
}
