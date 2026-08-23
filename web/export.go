package web

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Page is one concrete page of a static site.
type Page struct {
	// Path is the page URL, e.g. "/" or "/chapters/intro".
	Path string
	// Render writes the full page body.
	Render func(io.Writer) error
}

// Export writes pages and assets into outDir as a complete static website:
// the root page becomes outDir/index.html and every other page becomes
// outDir/<path>/index.html. Assets are written to their declared paths.
func Export(outDir string, pages []Page, assets map[string]string) error {
	sort.Slice(pages, func(i, j int) bool { return pages[i].Path < pages[j].Path })

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for _, p := range pages {
		dir := outDir
		if rel := strings.Trim(p.Path, "/"); rel != "" {
			dir = filepath.Join(outDir, filepath.FromSlash(rel))
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		f, err := os.Create(filepath.Join(dir, "index.html"))
		if err != nil {
			return err
		}
		if err := p.Render(f); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}

	for name, body := range assets {
		out := filepath.Join(outDir, filepath.FromSlash(cleanAssetPath(name)))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(out, []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// cleanAssetPath normalizes an asset URL into a safe relative path, refusing
// directory traversal.
func cleanAssetPath(name string) string {
	name = strings.TrimPrefix(name, "/")
	cleaned := path.Clean(name)
	if cleaned == "." || strings.HasPrefix(cleaned, "..") {
		return "assets/_"
	}
	return cleaned
}
