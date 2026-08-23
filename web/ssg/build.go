package ssg

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Build renders every page and writes the site into outDir. It returns the
// paths of the files created, relative to outDir.
func (s *Site) Build(outDir string) ([]string, error) {
	if err := s.prepare(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.used = map[string]bool{}
	s.mu.Unlock()

	var created []string
	for _, p := range s.pages {
		body, err := s.renderPage(p)
		if err != nil {
			return nil, err
		}
		rel := pageRelPath(p.Path)
		out := filepath.Join(outDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(out, []byte(body), 0o644); err != nil {
			return nil, err
		}
		created = append(created, rel)
	}

	if css := s.collectedCSS(); css != "" {
		rel := "assets/style.css"
		out := filepath.Join(outDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(out, []byte(css), 0o644); err != nil {
			return nil, err
		}
		created = append(created, rel)
	}

	names := make([]string, 0, len(s.assets))
	for name := range s.assets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		rel := cleanAssetPath(name)
		out := filepath.Join(outDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(out, []byte(s.assets[name]), 0o644); err != nil {
			return nil, err
		}
		created = append(created, rel)
	}
	return created, nil
}

// Handler returns an HTTP handler serving the site's pages and assets,
// rendered on demand for development.
func (s *Site) Handler() http.Handler {
	if err := s.prepare(); err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		})
	}
	mux := http.NewServeMux()
	for _, p := range s.pages {
		pp := p
		seen := map[string]bool{}
		paths := []string{p.Path}
		if !strings.HasSuffix(p.Path, ".html") {
			clean := strings.TrimSuffix(p.Path, "/")
			if slash := clean + "/"; !seen[slash] {
				paths = append(paths, slash)
			}
		}
		for _, rp := range paths {
			if seen[rp] {
				continue
			}
			seen[rp] = true
			mux.HandleFunc(rp, func(w http.ResponseWriter, r *http.Request) {
				body, err := s.renderPage(pp)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = io.WriteString(w, body)
			})
		}
	}
	for name, body := range s.assets {
		assetName, assetBody := name, body
		mux.HandleFunc("/"+assetName, func(w http.ResponseWriter, r *http.Request) {
			http.ServeContent(w, r, assetName, time.Time{}, strings.NewReader(assetBody))
		})
	}
	return mux
}

// RenderPage renders a page's root component inside its layout as a complete
// HTML document. It is the shared render path used by both live SSR and
// static export, guaranteeing identical output for identical input.
func (s *Site) RenderPage(p *Page) (string, error) {
	if err := s.prepare(); err != nil {
		return "", err
	}
	return s.renderPage(p)
}
