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
