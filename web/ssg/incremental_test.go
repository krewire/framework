package ssg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newIncSite() *Site {
	return New().
		Component(Component{Name: "Card", Body: `<div class="card">{{.Title}}</div>`}).
		Layout(Layout{Name: "Base", Body: `<html><body>{{template "Card" .}}</body></html>`}).
		Page(Page{Path: "/", Title: "Home", Layout: "Base", Root: "Card", Data: map[string]any{"Title": "A"}}).
		Page(Page{Path: "/about", Title: "About", Layout: "Base", Root: "Card", Data: map[string]any{"Title": "B"}})
}

func TestBuildIncremental_FullThenSkipUnchanged(t *testing.T) {
	out := t.TempDir()
	s := newIncSite()
	if _, err := s.BuildIncremental(out, false); err != nil {
		t.Fatal(err)
	}
	// Cache should now exist under .krewire/cache.
	if _, err := os.Stat(filepath.Join(out, "..", "cache", depGraphFile)); err != nil {
		t.Fatalf("cache not written: %v", err)
	}

	// Second run with no changes: nothing rewritten.
	s2 := newIncSite()
	created, err := s2.BuildIncremental(out, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 0 {
		t.Errorf("expected no files rewritten when unchanged, got %v", created)
	}
}

func TestBuildIncremental_ChangesOnlyAffectedPage(t *testing.T) {
	out := t.TempDir()
	s := newIncSite()
	if _, err := s.BuildIncremental(out, false); err != nil {
		t.Fatal(err)
	}

	// Change the shared Card component -> both pages depend on it -> both rebuild.
	s2 := New().
		Component(Component{Name: "Card", Body: `<div class="card">{{.Title}}!</div>`}).
		Layout(Layout{Name: "Base", Body: `<html><body>{{template "Card" .}}</body></html>`}).
		Page(Page{Path: "/", Title: "Home", Layout: "Base", Root: "Card", Data: map[string]any{"Title": "A"}}).
		Page(Page{Path: "/about", Title: "About", Layout: "Base", Root: "Card", Data: map[string]any{"Title": "B"}})
	created, err := s2.BuildIncremental(out, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 2 {
		t.Errorf("changing shared component should rebuild both pages, got %v", created)
	}
}

func TestBuildIncremental_ForceRebuildsAll(t *testing.T) {
	out := t.TempDir()
	s := newIncSite()
	if _, err := s.BuildIncremental(out, false); err != nil {
		t.Fatal(err)
	}
	s2 := newIncSite()
	created, err := s2.BuildIncremental(out, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 2 {
		t.Errorf("--force should rebuild all pages, got %v", created)
	}
}

func TestBuildIncremental_DeletedPageRemoved(t *testing.T) {
	out := t.TempDir()
	s := newIncSite()
	if _, err := s.BuildIncremental(out, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "about.html")); err != nil {
		t.Fatal("about.html should exist")
	}
	// Rebuild with only the home page.
	s2 := New().
		Component(Component{Name: "Card", Body: `<div class="card">{{.Title}}</div>`}).
		Layout(Layout{Name: "Base", Body: `<html><body>{{template "Card" .}}</body></html>`}).
		Page(Page{Path: "/", Title: "Home", Layout: "Base", Root: "Card", Data: map[string]any{"Title": "A"}})
	if _, err := s2.BuildIncremental(out, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "about.html")); !os.IsNotExist(err) {
		t.Error("deleted page output should have been removed")
	}
}

func TestBuildIncremental_AssetChangeInvalidatesPages(t *testing.T) {
	out := t.TempDir()
	s := New().
		Component(Component{Name: "Card", Body: `<div class="card">{{asset "assets/brand.css"}}</div>`}).
		Layout(Layout{Name: "Base", Body: `<html><body>{{template "Card" .}}</body></html>`}).
		Page(Page{Path: "/", Title: "Home", Layout: "Base", Root: "Card"}).
		Asset("assets/brand.css", "a{color:red}")
	if _, err := s.BuildIncremental(out, false); err != nil {
		t.Fatal(err)
	}

	// Change the referenced asset -> home page must rebuild.
	s2 := New().
		Component(Component{Name: "Card", Body: `<div class="card">{{asset "assets/brand.css"}}</div>`}).
		Layout(Layout{Name: "Base", Body: `<html><body>{{template "Card" .}}</body></html>`}).
		Page(Page{Path: "/", Title: "Home", Layout: "Base", Root: "Card"}).
		Asset("assets/brand.css", "a{color:blue}")
	created, err := s2.BuildIncremental(out, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(created, " "), "index.html") {
		t.Errorf("asset change should invalidate dependent page, got %v", created)
	}
}
