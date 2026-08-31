package ssg

import (
	"encoding/json"
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
	// Run the asset pipeline first so fingerprinted assets and the manifest
	// exist before templates (and the asset() helper) are prepared.
	if len(s.pipeline) > 0 {
		res, err := RunPipeline(s.assets, s.pipeline)
		if err != nil {
			return nil, err
		}
		s.assets = res.Assets
		s.manifest = res.Manifest
	}

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

	created = append(created, s.writeAssets(outDir)...)

	return created, nil
}

// BuildIncremental renders only the pages whose dependency signatures changed
// since the last build, persisting a dependency graph to .krewire/cache for
// fast dev loops and CI (KWF-99A63). When force is true, or the cache is
// missing/corrupt/theme-changed, it performs a full rebuild. Deleted pages have
// their output removed.
func (s *Site) BuildIncremental(outDir string, force bool) ([]string, error) {
	if len(s.pipeline) > 0 {
		res, err := RunPipeline(s.assets, s.pipeline)
		if err != nil {
			return nil, err
		}
		s.assets = res.Assets
		s.manifest = res.Manifest
	}
	if err := s.prepare(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.used = map[string]bool{}
	s.mu.Unlock()

	cacheDir := filepath.Join(filepath.Dir(outDir), "cache")
	prev := loadDepGraph(cacheDir)
	themeHash := s.themeHash()
	rebuildAll := force || prev == nil || prev.Version != depGraphVersion || prev.ThemeHash != themeHash

	var created []string
	newRoutes := make(map[string]bool, len(s.pages))
	for _, p := range s.pages {
		newRoutes[p.Path] = true
		rel := pageRelPath(p.Path)
		deps, err := s.pageDeps(p)
		if err != nil {
			return nil, err
		}
		sig := pageSig(deps, themeHash)
		skip := false
		if !rebuildAll {
			if node, ok := prev.Pages[p.Path]; ok && node.Sig == sig && node.OutRel == rel {
				if fileExists(filepath.Join(outDir, filepath.FromSlash(rel))) {
					skip = true
				}
			}
		}
		if skip {
			continue
		}
		body, err := s.renderPage(p)
		if err != nil {
			return nil, err
		}
		out := filepath.Join(outDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(out, []byte(body), 0o644); err != nil {
			return nil, err
		}
		created = append(created, rel)
	}

	// Remove output of pages that no longer exist.
	if prev != nil {
		for route, node := range prev.Pages {
			if !newRoutes[route] {
				_ = os.Remove(filepath.Join(outDir, filepath.FromSlash(node.OutRel)))
			}
		}
	}

	created = append(created, s.writeAssets(outDir)...)

	g := &depGraph{Version: depGraphVersion, ThemeHash: themeHash, Pages: map[string]*pageNode{}}
	for _, p := range s.pages {
		deps, _ := s.pageDeps(p)
		g.Pages[p.Path] = &pageNode{OutRel: pageRelPath(p.Path), Sig: pageSig(deps, themeHash)}
	}
	if err := g.save(cacheDir); err != nil {
		return nil, err
	}
	return created, nil
}

// writeAssets emits the collected CSS, every registered asset, and the asset
// manifest. It is shared by Build and BuildIncremental.
func (s *Site) writeAssets(outDir string) []string {
	var created []string
	if css := s.collectedCSS(); css != "" {
		rel := "assets/style.css"
		out := filepath.Join(outDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err == nil {
			if werr := os.WriteFile(out, []byte(css), 0o644); werr == nil {
				created = append(created, rel)
			}
		}
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
			return created
		}
		if err := os.WriteFile(out, []byte(s.assets[name]), 0o644); err != nil {
			return created
		}
		created = append(created, rel)
	}
	if len(s.manifest) > 0 {
		if mb, err := json.MarshalIndent(s.manifest, "", "  "); err == nil {
			rel := "manifest.json"
			out := filepath.Join(outDir, rel)
			if werr := os.WriteFile(out, append(mb, '\n'), 0o644); werr == nil {
				created = append(created, rel)
			}
		}
	}
	return created
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
		mux.HandleFunc(p.Path, func(w http.ResponseWriter, r *http.Request) {
			body, err := s.renderPage(pp)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.WriteString(w, body)
		})
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
