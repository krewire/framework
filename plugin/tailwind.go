package plugin

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

func init() {
	Register(&Tailwind{})
}

// Tailwind implements Plugin for Tailwind CSS via the Tailwind CLI.
// Detection: presence of tailwind.config.js at the project root.
// Build: runs `npx tailwindcss -i assets/tailwind.css -o <outDir>/assets/tailwind.css --minify`
// (or ./node_modules/.bin/tailwindcss if npx is not available). Failures are
// logged as warnings and do not fail the overall site build — Tailwind is
// an optional plugin, not a core requirement.
type Tailwind struct{}

func (t *Tailwind) Name() string { return "tailwind" }

func (t *Tailwind) Aliases() []string { return []string{"twcss", "tailwindcss"} }

func (t *Tailwind) Detect(root string) bool {
	for _, name := range []string{"tailwind.config.js", "tailwind.config.cjs", "tailwind.config.ts"} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return true
		}
	}
	return false
}

func (t *Tailwind) Build(root, outDir string) error {
	input := filepath.Join(root, "assets", "tailwind.css")
	if _, err := os.Stat(input); err != nil {
		slog.Warn("tailwind: assets/tailwind.css not found, skipping", "root", root)
		return nil
	}
	output := filepath.Join(outDir, "assets", "tailwind.css")
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return fmt.Errorf("tailwind: mkdir: %w", err)
	}

	// Prefer local binary, fall back to npx
	candidates := [][]string{
		{"./node_modules/.bin/tailwindcss", "-i", input, "-o", output, "--minify"},
		{"npx", "tailwindcss", "-i", input, "-o", output, "--minify"},
	}
	var lastErr error
	for _, args := range candidates {
		bin := args[0]
		if _, err := exec.LookPath(bin); err != nil && bin != "npx" {
			continue
		}
		slog.Info("tailwind: building", "input", input, "output", output, "cmd", args)
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("tailwind: %v: %s", err, string(out))
			slog.Warn("tailwind: build failed, skipping", "err", lastErr)
			continue
		}
		slog.Info("tailwind: built", "output", output)
		return nil
	}
	if lastErr != nil {
		slog.Warn("tailwind: CLI not available, skipping", "err", lastErr, "hint", "npm install -D tailwindcss")
	}
	return nil
}

const tailwindConfigTemplate = `/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["pages/**/*.kiw","components/**/*.kiw","layouts/**/*.kiw","content/**/*.md"],
  theme: {
    extend: {
      colors: {
        primary: "var(--color-primary)",
      },
    },
  },
  plugins: [],
}
`

const tailwindInputTemplate = `@tailwind base;
@tailwind components;
@tailwind utilities;
`

// Add installs tailwind at version ("" or "latest" means latest) — part of scalable plugin system.
func (t *Tailwind) Add(root, version string) error {
	if version == "" || version == "latest" {
		version = "latest"
	}
	pkg := "tailwindcss@" + version
	if version == "latest" {
		pkg = "tailwindcss@latest"
	}
	slog.Info("tailwind: installing", "package", pkg, "root", root)
	if err := ensureTailwindConfig(root); err != nil {
		return err
	}
	if err := ensureTailwindInput(root); err != nil {
		return err
	}
	if err := npmInstall(root, pkg, true); err != nil {
		slog.Warn("tailwind: npm install failed, config files still created", "err", err, "hint", "run npm install -D "+pkg+" manually")
		return err
	}
	slog.Info("tailwind: installed", "package", pkg)
	return nil
}

// Remove uninstalls tailwind.
func (t *Tailwind) Remove(root string) error {
	slog.Info("tailwind: removing", "root", root)
	removed := 0
	for _, name := range []string{"tailwind.config.js", "tailwind.config.cjs", "tailwind.config.ts"} {
		p := filepath.Join(root, name)
		if _, err := os.Stat(p); err == nil {
			if err := os.Remove(p); err != nil {
				return fmt.Errorf("tailwind: remove %s: %w", name, err)
			}
			removed++
			slog.Info("tailwind: removed", "file", name)
		}
	}
	// do not delete assets/tailwind.css if user customized — only if it matches template
	input := filepath.Join(root, "assets", "tailwind.css")
	if data, err := os.ReadFile(input); err == nil {
		if string(data) == tailwindInputTemplate {
			_ = os.Remove(input)
			slog.Info("tailwind: removed", "file", "assets/tailwind.css")
		} else {
			slog.Info("tailwind: kept customized assets/tailwind.css")
		}
	}
	if err := npmUninstall(root, "tailwindcss"); err != nil {
		slog.Warn("tailwind: npm uninstall failed", "err", err)
		return err
	}
	if removed == 0 {
		slog.Info("tailwind: already not installed")
	}
	return nil
}

func ensureTailwindConfig(root string) error {
	for _, name := range []string{"tailwind.config.js", "tailwind.config.cjs", "tailwind.config.ts"} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return nil
		}
	}
	path := filepath.Join(root, "tailwind.config.js")
	if err := os.WriteFile(path, []byte(tailwindConfigTemplate), 0o644); err != nil {
		return fmt.Errorf("tailwind: write config: %w", err)
	}
	slog.Info("tailwind: created", "file", "tailwind.config.js")
	return nil
}

func ensureTailwindInput(root string) error {
	path := filepath.Join(root, "assets", "tailwind.css")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("tailwind: mkdir assets: %w", err)
	}
	if err := os.WriteFile(path, []byte(tailwindInputTemplate), 0o644); err != nil {
		return fmt.Errorf("tailwind: write input: %w", err)
	}
	slog.Info("tailwind: created", "file", "assets/tailwind.css")
	return nil
}

func npmInstall(root, pkg string, dev bool) error {
	args := []string{"install", pkg}
	if dev {
		args = []string{"install", "-D", pkg}
	}
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm not found in PATH")
	}
	cmd := exec.Command("npm", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install %s: %v: %s", pkg, err, string(out))
	}
	return nil
}

func npmUninstall(root, pkg string) error {
	if _, err := exec.LookPath("npm"); err != nil {
		return fmt.Errorf("npm not found in PATH")
	}
	cmd := exec.Command("npm", "uninstall", pkg)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm uninstall %s: %v: %s", pkg, err, string(out))
	}
	return nil
}
