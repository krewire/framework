package ssg

import (
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Open implements fs.FS over the site's asset map, letting an App serve assets.
func (s *Site) Open(name string) (fs.File, error) {
	name = strings.TrimPrefix(name, "/")
	body, ok := s.assets[name]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return fsFile{name: name, r: strings.NewReader(body)}, nil
}

type fsFile struct {
	name string
	r    *strings.Reader
}

func (f fsFile) Stat() (fs.FileInfo, error) {
	return fi{name: f.name, size: int64(f.r.Len())}, nil
}
func (f fsFile) Read(p []byte) (int, error) { return f.r.Read(p) }
func (f fsFile) Close() error               { return nil }

type fi struct {
	name string
	size int64
}

func (i fi) Name() string       { return path.Base(i.name) }
func (i fi) Size() int64        { return i.size }
func (i fi) Mode() fs.FileMode  { return 0o444 }
func (i fi) ModTime() time.Time { return time.Time{} }
func (i fi) IsDir() bool        { return false }
func (i fi) Sys() any           { return nil }

// pageRelPath maps a page route to a sibling .html file: "/" -> "index.html",
// "/about" -> "about.html", "/docs/quickstart" -> "docs/quickstart.html".
func pageRelPath(p string) string {
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return "index.html"
	}
	if strings.HasSuffix(p, ".html") {
		return p
	}
	return filepath.FromSlash(p) + ".html"
}

// cleanAssetPath normalizes an asset path, refusing directory traversal.
func cleanAssetPath(name string) string {
	name = strings.TrimPrefix(name, "/")
	cleaned := path.Clean(name)
	if cleaned == "." || strings.HasPrefix(cleaned, "..") {
		return "assets/_"
	}
	return cleaned
}
