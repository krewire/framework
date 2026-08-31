package ssg

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// depGraphVersion is the on-disk format version for the incremental build
// cache (KWF-99A63). A version mismatch forces a full rebuild.
const depGraphVersion = 1

const depGraphFile = "depgraph.json"

// depGraph persists the per-page dependency signatures between builds so an
// incremental Build only re-renders pages whose inputs changed.
type depGraph struct {
	Version   int                  `json:"version"`
	ThemeHash string               `json:"themeHash"`
	Pages     map[string]*pageNode `json:"pages"`
}

// pageNode records the output path and a content signature for one page.
type pageNode struct {
	OutRel string `json:"outRel"`
	Sig    string `json:"sig"`
}

// refRe matches component/mount/asset references inside a template body so the
// dependency graph can follow component calls and asset usage.
var refRe = regexp.MustCompile(`\b(component|mount|asset)\s+"([^"]+)"`)

func hashBytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// themeHash captures the site-wide scoped CSS (which embeds theme variables),
// used to invalidate every page when the theme or config changes.
func (s *Site) themeHash() string {
	return hashBytes([]byte(s.collectedCSS()))
}

// pageDeps computes the dependency signature inputs for a page: its layout and
// root component bodies, every component reached through component/mount calls
// (recursively), the content of any asset referenced via asset(), and a hash
// of the page's data frontier (which includes Markdown content).
func (s *Site) pageDeps(p *Page) (map[string]string, error) {
	deps := map[string]string{}
	queue := []string{}
	if p.Root != "" {
		queue = append(queue, p.Root)
	}
	seen := map[string]bool{}
	addBody := func(b string) {
		for _, sm := range refRe.FindAllStringSubmatch(b, -1) {
			switch sm[1] {
			case "component", "mount":
				if !seen[sm[2]] {
					queue = append(queue, sm[2])
				}
			case "asset":
				if body, ok := s.assets[sm[2]]; ok {
					deps["asset:"+sm[2]] = hashBytes([]byte(body))
				}
			}
		}
	}
	if l, ok := s.layouts[p.Layout]; ok {
		addBody(l.Body)
	}
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		if seen[name] {
			continue
		}
		seen[name] = true
		c, ok := s.comps[name]
		if !ok {
			continue
		}
		deps["comp:"+name] = hashBytes([]byte(c.Body))
		addBody(c.Body)
	}
	db, err := json.Marshal(p.Data)
	if err != nil {
		db = []byte(fmt.Sprintf("%v", p.Data))
	}
	deps["data:"+p.Path] = hashBytes(db)
	return deps, nil
}

// pageSig produces a stable signature for a page from its dependency hashes and
// the site theme hash. Identical inputs always yield the same signature.
func pageSig(deps map[string]string, themeHash string) string {
	keys := make([]string, 0, len(deps))
	for k := range deps {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	h.Write([]byte(themeHash))
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte(deps[k]))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// loadDepGraph reads the persisted graph from cacheDir. A missing or corrupt
// cache (wrong version, bad JSON) returns nil, forcing a full rebuild.
func loadDepGraph(cacheDir string) *depGraph {
	b, err := os.ReadFile(filepath.Join(cacheDir, depGraphFile))
	if err != nil {
		return nil
	}
	var g depGraph
	if err := json.Unmarshal(b, &g); err != nil || g.Version != depGraphVersion {
		return nil
	}
	if g.Pages == nil {
		g.Pages = map[string]*pageNode{}
	}
	return &g
}

func (g *depGraph) save(cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cacheDir, depGraphFile), append(b, '\n'), 0o644)
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
