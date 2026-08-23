// Package assets unifies static asset and resource sources into one
// namespace, serves them over HTTP with caching semantics, and provides
// content-hash fingerprinting with a template-facing manifest.
//
// A Store mounts any fs.FS — an os.DirFS directory or a //go:embed value —
// so shipped resources (configs, seeds, i18n documents) and public assets
// share one lookup path. First mounted source that contains the name wins.
package assets

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Store is a searchable set of asset sources.
type Store struct {
	sources []fs.FS
}

// NewStore returns an empty Store.
func NewStore() *Store { return &Store{} }

// Mount adds fsys as a source. Earlier mounts win on name conflicts.
func (s *Store) Mount(fsys fs.FS) *Store {
	s.sources = append(s.sources, fsys)
	return s
}

// MountDir adds the directory dir as a source.
func (s *Store) MountDir(dir string) *Store {
	return s.Mount(os.DirFS(dir))
}

// MountEmbed adds an //go:embed FS as a source. Names keep their embedded
// layout; mount a sub-FS via fs.Sub to namespace it.
func (s *Store) MountEmbed(e embed.FS) *Store {
	return s.Mount(e)
}

// Open returns the raw content, detected content type, and strong ETag for
// name. The first source containing the file wins.
func (s *Store) Open(name string) ([]byte, string, string, error) {
	name = cleanName(name)
	if name == "" {
		return nil, "", "", fmt.Errorf("assets: empty name")
	}
	for _, src := range s.sources {
		b, err := fs.ReadFile(src, name)
		if err == nil {
			return b, ContentType(name), ETag(b), nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, "", "", fmt.Errorf("assets: read %s: %w", name, err)
		}
	}
	return nil, "", "", fmt.Errorf("assets: %s: %w", name, fs.ErrNotExist)
}

// Names lists every asset across all sources, sorted, deduplicated.
func (s *Store) Names() ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, src := range s.sources {
		err := fs.WalkDir(src, ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			p = cleanName(p)
			if p != "" && !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(out)
	return out, nil
}

// Fingerprint returns the cache-busting form of name: "<base>.<hash8>.<ext>".
func (s *Store) Fingerprint(name string) (string, error) {
	b, _, _, err := s.Open(name)
	if err != nil {
		return "", err
	}
	return FingerprintName(name, b), nil
}

// Manifest maps every known asset to its fingerprinted path. Marshal it to
// JSON and hand it to templates for immutable-cache URLs.
func (s *Store) Manifest() (map[string]string, error) {
	names, err := s.Names()
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(names))
	for _, n := range names {
		b, _, _, err := s.Open(n)
		if err != nil {
			return nil, err
		}
		out[n] = FingerprintName(n, b)
	}
	return out, nil
}

// Handler serves GET/HEAD requests resolving names against the store with
// revalidation-friendly caching (ETag + must-revalidate).
func (s *Store) Handler() http.Handler { return s.handler(false) }

// HandlerImmutable serves like Handler but assumes fingerprinted URLs and
// emits long-lived immutable caching.
func (s *Store) HandlerImmutable() http.Handler { return s.handler(true) }

func (s *Store) handler(immutable bool) http.Handler {
	cache := "public, max-age=0, must-revalidate"
	if immutable {
		cache = "public, max-age=31536000, immutable"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		b, ctype, etag, err := s.Open(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", ctype)
		w.Header().Set("ETag", etag)
		w.Header().Set("Cache-Control", cache)
		if match := r.Header.Get("If-None-Match"); match != "" && etagMatches(match, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if r.Method == http.MethodHead {
			w.Header().Set("Content-Length", strconv.Itoa(len(b)))
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(b)))
		_, _ = w.Write(b)
	})
}

// JSON decodes an embedded JSON resource into T.
func JSON[T any](s *Store, name string) (*T, error) {
	b, _, _, err := s.Open(name)
	if err != nil {
		return nil, err
	}
	out := new(T)
	if err := json.Unmarshal(b, out); err != nil {
		return nil, fmt.Errorf("assets: %s: %w", name, err)
	}
	return out, nil
}

// YAML decodes an embedded YAML resource into T.
func YAML[T any](s *Store, name string) (*T, error) {
	b, _, _, err := s.Open(name)
	if err != nil {
		return nil, err
	}
	out := new(T)
	if err := yaml.Unmarshal(b, out); err != nil {
		return nil, fmt.Errorf("assets: %s: %w", name, err)
	}
	return out, nil
}

// ContentType maps a file extension onto its MIME type, defaulting to
// application/octet-stream.
func ContentType(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".css":
		return "text/css; charset=utf-8"
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".json", ".map", ".webmanifest":
		return "application/json; charset=utf-8"
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".md":
		return "text/markdown; charset=utf-8"
	case ".xml":
		return "application/xml; charset=utf-8"
	case ".woff2":
		return "font/woff2"
	case ".woff":
		return "font/woff"
	case ".ttf":
		return "font/ttf"
	case ".ico":
		return "image/x-icon"
	}
	if t := mime.TypeByExtension(strings.ToLower(path.Ext(name))); t != "" {
		return t
	}
	return "application/octet-stream"
}

// ETag returns a strong entity tag for content.
func ETag(content []byte) string {
	sum := sha256.Sum256(content)
	return `"` + hex.EncodeToString(sum[:16]) + `"`
}

// FingerprintName derives "<base>.<hash8>.<ext>" from name and content.
func FingerprintName(name string, content []byte) string {
	sum := sha256.Sum256(content)
	hash := hex.EncodeToString(sum[:4])
	ext := path.Ext(name)
	base := strings.TrimSuffix(name, ext)
	return base + "." + hash + ext
}

func etagMatches(header, etag string) bool {
	for _, part := range strings.Split(header, ",") {
		if strings.TrimSpace(part) == etag || strings.TrimSpace(part) == "*" {
			return true
		}
	}
	return false
}

// cleanName normalizes a request path into an fs.FS name.
func cleanName(name string) string {
	name = path.Clean("/" + strings.ReplaceAll(name, "\\", "/"))
	return strings.TrimPrefix(name, "/")
}
