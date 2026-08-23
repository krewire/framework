package ssg

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"gopkg.in/yaml.v3"
)

// ContentConfig holds frontmatter fields for content pages.
type ContentConfig struct {
	Title       string   `yaml:"title"`
	Date        string   `yaml:"date"`
	Draft       bool     `yaml:"draft"`
	Tags        []string `yaml:"tags"`
	Description string   `yaml:"description"`
	Slug        string   `yaml:"slug"`
	Layout      string   `yaml:"layout"`
	// Additional fields are captured in Extra
	Extra map[string]any `yaml:",inline"`
}

// CollectionConfig defines a content collection in ssg.yaml.
type CollectionConfig struct {
	Name    string            `yaml:"name"`
	Dir     string            `yaml:"dir"`
	Pattern string            `yaml:"pattern"`
	Layout  string            `yaml:"layout"`
	Output  string            `yaml:"output"`
	Schema  *CollectionSchema `yaml:"schema"`
}

// CollectionSchema validates frontmatter fields.
type CollectionSchema struct {
	Required []string `yaml:"required"`
	Optional []string `yaml:"optional"`
}

// ParsedContent holds the parsed result of a Markdown file.
type ParsedContent struct {
	Config  ContentConfig
	Content template.HTML
	Path    string
	URL     string
	RawDate time.Time
}

// Collection holds all pages in a collection.
type Collection struct {
	Name   string
	Pages  []*ParsedContent
	Config CollectionConfig
}

// markdownParser is the shared markdown parser.
var markdownParser = parser.NewWithExtensions(
	parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock,
)

// htmlRenderer is the shared HTML renderer.
var htmlRenderer = html.NewRenderer(html.RendererOptions{
	Flags: html.CommonFlags | html.HrefTargetBlank,
})

// ParseContent parses a Markdown file with YAML frontmatter.
func ParseContent(filePath string, baseDir string) (*ParsedContent, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// Split frontmatter and content
	parts := strings.SplitN(string(data), "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("content: missing frontmatter in %s", filePath)
	}

	var cfg ContentConfig
	if err := yaml.Unmarshal([]byte(parts[1]), &cfg); err != nil {
		return nil, fmt.Errorf("content: frontmatter parse error in %s: %w", filePath, err)
	}

	// Parse date
	var rawDate time.Time
	if cfg.Date != "" {
		rawDate, err = parseFlexibleDate(cfg.Date)
		if err != nil {
			return nil, fmt.Errorf("content: invalid date in %s: %w", filePath, err)
		}
	}

	// Render markdown to HTML
	md := []byte(strings.TrimSpace(parts[2]))
	doc := markdownParser.Parse(md)
	htmlBytes := markdown.Render(doc, htmlRenderer)

	// Determine slug and URL
	rel, _ := filepath.Rel(baseDir, filePath)
	slug := cfg.Slug
	if slug == "" {
		slug = strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
	}
	url := "/" + strings.TrimSuffix(rel, filepath.Ext(rel)) + "/"

	return &ParsedContent{
		Config:  cfg,
		Content: template.HTML(htmlBytes),
		Path:    filePath,
		URL:     url,
		RawDate: rawDate,
	}, nil
}

// parseFlexibleDate parses common date formats.
func parseFlexibleDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"Jan 2, 2006",
		"January 2, 2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date format: %s", s)
}

// BuildCollection scans a directory for Markdown files and returns a Collection.
func BuildCollection(cc CollectionConfig, siteRoot string) (*Collection, error) {
	baseDir := filepath.Join(siteRoot, cc.Dir)
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return &Collection{Name: cc.Name, Config: cc}, nil // empty collection
	}

	pattern := cc.Pattern
	if pattern == "" {
		pattern = "*.md"
	}

	var pages []*ParsedContent
	err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !matchesPattern(d.Name(), pattern) {
			return nil
		}
		pc, err := ParseContent(path, baseDir)
		if err != nil {
			return fmt.Errorf("collection %s: %w", cc.Name, err)
		}
		pages = append(pages, pc)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort by date descending (newest first)
	sort.Slice(pages, func(i, j int) bool {
		if pages[i].RawDate.IsZero() && pages[j].RawDate.IsZero() {
			return pages[i].URL < pages[j].URL
		}
		if pages[i].RawDate.IsZero() {
			return false
		}
		if pages[j].RawDate.IsZero() {
			return true
		}
		return pages[i].RawDate.After(pages[j].RawDate)
	})

	return &Collection{
		Name:   cc.Name,
		Pages:  pages,
		Config: cc,
	}, nil
}

// matchesPattern checks if a filename matches a simple glob pattern.
func matchesPattern(name, pattern string) bool {
	if pattern == "" || pattern == "*" {
		return strings.HasSuffix(name, ".md")
	}
	if strings.HasPrefix(pattern, "*.") {
		ext := pattern[1:]
		return strings.HasSuffix(name, ext)
	}
	matched, _ := filepath.Match(pattern, name)
	return matched
}

// FilterDrafts removes draft pages unless includeDrafts is true.
func (c *Collection) FilterDrafts(includeDrafts bool) {
	if includeDrafts {
		return
	}
	filtered := c.Pages[:0]
	for _, p := range c.Pages {
		if !p.Config.Draft {
			filtered = append(filtered, p)
		}
	}
	c.Pages = filtered
}

// FilterFuture removes future-dated pages unless includeFuture is true.
func (c *Collection) FilterFuture(includeFuture bool) {
	if includeFuture {
		return
	}
	now := time.Now()
	filtered := c.Pages[:0]
	for _, p := range c.Pages {
		if p.RawDate.IsZero() || !p.RawDate.After(now) {
			filtered = append(filtered, p)
		}
	}
	c.Pages = filtered
}

// GenerateOutputPath computes the output path for a page using the collection's output pattern.
func (c *Collection) GenerateOutputPath(p *ParsedContent) string {
	out := c.Config.Output
	if out == "" {
		return p.URL
	}
	// Replace placeholders
	out = strings.ReplaceAll(out, ":collection", c.Name)
	out = strings.ReplaceAll(out, ":slug", slugify(p.Config.Slug))
	if !p.RawDate.IsZero() {
		out = strings.ReplaceAll(out, ":year", fmt.Sprintf("%04d", p.RawDate.Year()))
		out = strings.ReplaceAll(out, ":month", fmt.Sprintf("%02d", p.RawDate.Month()))
		out = strings.ReplaceAll(out, ":day", fmt.Sprintf("%02d", p.RawDate.Day()))
	}
	// Ensure trailing slash for clean URLs
	if !strings.HasSuffix(out, "/") && !strings.Contains(out, ".") {
		out += "/"
	}
	return "/" + strings.TrimPrefix(out, "/")
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	// Remove non-alphanumeric except hyphen
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// markdownFunc is the template helper for inline markdown rendering.
func markdownFunc(s string) template.HTML {
	doc := markdownParser.Parse([]byte(s))
	return template.HTML(markdown.Render(doc, htmlRenderer))
}
