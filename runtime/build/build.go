// Package build produces a Krewire WASM module plus JS glue from a component
// entry point (KWF-T4X9P FRK-WASM-001/002/003/004). It wraps
// "go build GOOS=js GOARCH=wasm" with log capture, content-hashing, and
// actionable diagnostics for toolchain and version errors.
package build

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Artifacts holds the paths and digest of a WASM build (FRK-WASM-001).
type Artifacts struct {
	WASM   string // path to the compiled .wasm module
	JS     string // path to the JS glue (wasm_exec.js)
	Digest string // sha256 of the .wasm file, hex-encoded
}

// Config drives a WASM build (FRK-WASM-001).
type Config struct {
	// Entry is the main package path to build (e.g. "./frontend").
	Entry string
	// OutDir receives the .wasm module and copied JS glue.
	OutDir string
	// Name is the output module name (default "runtime").
	Name string
	// GoRoot overrides the Go toolchain location (empty = lookup).
	GoRoot string
}

// BuildWASM compiles the entry package to WASM and returns the artifacts
// (FRK-WASM/001). The JS glue is copied from the Go distribution. On
// failure it returns a diagnostic error with actionable guidance
// (FRK-WASM/004).
func BuildWASM(cfg Config) (Artifacts, error) {
	if cfg.Entry == "" {
		return Artifacts{}, usageError("entry package path is required")
	}
	if cfg.OutDir == "" {
		return Artifacts{}, usageError("output directory is required")
	}
	name := cfg.Name
	if name == "" {
		name = "runtime"
	}

	goRoot, err := resolveGoRoot(cfg.GoRoot)
	if err != nil {
		return Artifacts{}, err
	}

	if err := os.MkdirAll(cfg.OutDir, 0o755); err != nil {
		return Artifacts{}, fmt.Errorf("build: create output dir: %w", err)
	}

	wasmPath := filepath.Join(cfg.OutDir, name+".wasm")
	var logBuf bytes.Buffer
	cmd := exec.Command(filepath.Join(goRoot, "bin", "go"), "build", "-o", wasmPath, cfg.Entry)
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	cmd.Stdout = &logBuf
	cmd.Stderr = &logBuf
	if err := cmd.Run(); err != nil {
		return Artifacts{}, diagnoseBuildFailure(goRoot, logBuf.String(), err)
	}

	digest, err := fileDigest(wasmPath)
	if err != nil {
		return Artifacts{}, err
	}

	jsPath, err := copyJSGlue(goRoot, cfg.OutDir)
	if err != nil {
		return Artifacts{}, err
	}

	return Artifacts{
		WASM:   wasmPath,
		JS:     jsPath,
		Digest: digest,
	}, nil
}

// fileDigest returns the hex-encoded sha256 of path (FRK-WASM-003).
func fileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("build: read wasm: %w", err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// Fingerprint returns a short, URL-safe digest prefix for cache-busting
// (FRK-WASM/003).
func Fingerprint(digest string) string {
	if len(digest) < 8 {
		return digest
	}
	return digest[:8]
}

func resolveGoRoot(override string) (string, error) {
	if override != "" {
		if _, err := os.Stat(filepath.Join(override, "bin", "go")); err != nil {
			return "", usageError("go toolchain not found at " + override + "/bin/go")
		}
		return override, nil
	}
	cmd := exec.Command("go", "env", "GOROOT")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("build: locate go toolchain: %w (set GoRoot explicitly)", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func copyJSGlue(goRoot, outDir string) (string, error) {
	src := filepath.Join(goRoot, "misc", "wasm", "wasm_exec.js")
	data, err := os.ReadFile(src)
	if err != nil {
		return "", fmt.Errorf("build: read wasm_exec.js: %w (go version may be incompatible)", err)
	}
	dst := filepath.Join(outDir, "wasm_exec.js")
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return "", fmt.Errorf("build: write wasm_exec.js: %w", err)
	}
	return dst, nil
}

func diagnoseBuildFailure(goRoot, logs string, err error) error {
	if strings.Contains(logs, "GOOS=js GOARCH=wasm") && strings.Contains(logs, "not recognized") {
		return fmt.Errorf("build: wasm target not supported by go %s: upgrade to go 1.21+ (details: %s)", goShortVersion(goRoot), logs)
	}
	if strings.Contains(logs, "missing go.sum entry") || strings.Contains(logs, "cannot find module") {
		return fmt.Errorf("build: module resolution failed: run \"go mod tidy\" in the entry package (details: %s)", logs)
	}
	return fmt.Errorf("build: wasm compile failed: %w (logs: %s)", err, logs)
}

func goShortVersion(goRoot string) string {
	cmd := exec.Command(filepath.Join(goRoot, "bin", "go"), "version")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func usageError(msg string) error {
	return fmt.Errorf("build: %s (exit 2)", msg)
}
