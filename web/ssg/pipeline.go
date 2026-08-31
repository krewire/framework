package ssg

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
)

// PipelineRule configures the transforms applied to assets whose path matches
// Glob. Use lists transformer names in order: copy, hash, minify-css,
// minify-js, resize, webp, avif.
type PipelineRule struct {
	// Glob selects source assets by path (e.g. "assets/*.css").
	Glob string `yaml:"glob"`
	// Use is the ordered list of transformers to apply.
	Use []string `yaml:"use"`
	// Width/Height are resize targets (pixels); 0 means "preserve".
	Width int `yaml:"width"`
	// Height is the resize height target.
	Height   int    `yaml:"height"`
	Fit      string `yaml:"fit"`     // contain|cover|fill
	Format   string `yaml:"format"`  // jpeg|png|webp|avif
	Quality  int    `yaml:"quality"` // 1-100
	Original *bool  `yaml:"original"`
}

// PipelineResult is the output of running the asset pipeline.
type PipelineResult struct {
	// Assets maps fingerprinted output paths (relative to outDir/assets/) to
	// their content.
	Assets map[string]string
	// Manifest maps a logical asset path to its fingerprinted URL
	// (/assets/...), consumed by the asset() template helper.
	Manifest map[string]string
}

// RunPipeline transforms source assets per rules (KWF-DR5YU). Assets not
// matching any rule are copied verbatim. The output is deterministic: assets
// are processed in sorted path order.
func RunPipeline(assets map[string]string, rules []PipelineRule) (*PipelineResult, error) {
	res := &PipelineResult{Assets: map[string]string{}, Manifest: map[string]string{}}
	names := make([]string, 0, len(assets))
	for n := range assets {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		body := assets[name]
		rule, ok := matchRule(name, rules)
		if !ok {
			res.Assets[name] = body
			res.Manifest[name] = assetURL(name)
			continue
		}
		outs, err := transform(name, []byte(body), rule)
		if err != nil {
			return nil, fmt.Errorf("pipeline %q: %w", name, err)
		}
		for _, o := range outs {
			res.Assets[o.Path] = string(o.Content)
		}
		if len(outs) > 0 {
			res.Manifest[name] = assetURL(outs[0].Path)
		}
	}

	// Rewrite CSS url() references to fingerprinted siblings (FRK-AS-033).
	for name, body := range res.Assets {
		if !strings.EqualFold(filepath.Ext(name), ".css") {
			continue
		}
		res.Assets[name] = rewriteCSSURLs(body, res.Manifest)
	}
	return res, nil
}

func matchRule(name string, rules []PipelineRule) (PipelineRule, bool) {
	for _, r := range rules {
		if r.Glob == "" {
			continue
		}
		if ok, _ := filepath.Match(r.Glob, name); ok {
			return r, true
		}
	}
	return PipelineRule{}, false
}

type assetOutput struct {
	Path    string
	Content []byte
}

func transform(name string, body []byte, rule PipelineRule) ([]assetOutput, error) {
	hashIt := contains(rule.Use, "hash")
	keepOriginal := rule.Original == nil || *rule.Original

	base := body
	if contains(rule.Use, "minify-css") {
		out, err := minifyBytes("text/css", base)
		if err != nil {
			return nil, fmt.Errorf("minify-css: %w", err)
		}
		base = out
	}
	if contains(rule.Use, "minify-js") {
		out, err := minifyBytes("application/javascript", base)
		if err != nil {
			return nil, fmt.Errorf("minify-js: %w", err)
		}
		base = out
	}
	if contains(rule.Use, "resize") {
		img, err := imaging.Decode(bytes.NewReader(base))
		if err != nil {
			return nil, fmt.Errorf("resize decode: %w", err)
		}
		img = resizeImage(img, rule)
		var buf bytes.Buffer
		enc := imaging.JPEG
		if strings.EqualFold(filepath.Ext(name), ".png") {
			enc = imaging.PNG
		}
		if err := imaging.Encode(&buf, img, enc); err != nil {
			return nil, fmt.Errorf("resize encode: %w", err)
		}
		base = buf.Bytes()
	}

	var outs []assetOutput
	if keepOriginal {
		p := name
		if hashIt {
			p = fingerprint(p, base)
		}
		outs = append(outs, assetOutput{Path: p, Content: base})
	}
	if contains(rule.Use, "webp") {
		out, err := encodeWebp(base, rule.Quality)
		if err != nil {
			return nil, fmt.Errorf("webp: %w", err)
		}
		p := swapExt(name, ".webp")
		if hashIt {
			p = fingerprint(p, out)
		}
		outs = append(outs, assetOutput{Path: p, Content: out})
	}
	if contains(rule.Use, "avif") {
		return nil, fmt.Errorf("avif encoding requires an external encoder dependency (not bundled; see KWF-DR5YU follow-up)")
	}
	return outs, nil
}

func minifyBytes(mediaType string, b []byte) ([]byte, error) {
	m := minify.New()
	switch mediaType {
	case "text/css":
		m.Add("text/css", &css.Minifier{})
	case "application/javascript":
		m.Add("application/javascript", &js.Minifier{})
	default:
		return b, nil
	}
	out, err := m.Bytes(mediaType, b)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func resizeImage(img image.Image, r PipelineRule) image.Image {
	if r.Width <= 0 && r.Height <= 0 {
		return img
	}
	switch strings.ToLower(r.Fit) {
	case "contain":
		if r.Width > 0 && r.Height > 0 {
			return imaging.Fit(img, r.Width, r.Height, imaging.Lanczos)
		}
	case "cover", "fill":
		if r.Width > 0 && r.Height > 0 {
			return imaging.Fill(img, r.Width, r.Height, imaging.Center, imaging.Lanczos)
		}
	}
	if r.Height <= 0 {
		return imaging.Resize(img, r.Width, 0, imaging.Lanczos)
	}
	if r.Width <= 0 {
		return imaging.Resize(img, 0, r.Height, imaging.Lanczos)
	}
	return imaging.Resize(img, r.Width, r.Height, imaging.Lanczos)
}

func encodeWebp(b []byte, quality int) ([]byte, error) {
	img, err := imaging.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("webp decode: %w", err)
	}
	if quality <= 0 || quality > 100 {
		quality = 80
	}
	var buf bytes.Buffer
	if err := webp.Encode(&buf, img, &webp.Options{Lossless: false, Quality: float32(quality), Exact: true}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// fingerprint appends the first 8 hex chars of the SHA256 of content before
// the file extension: "style.css" -> "style.a1b2c3d4.css".
func fingerprint(p string, content []byte) string {
	sum := sha256.Sum256(content)
	h := hex.EncodeToString(sum[:])[:8]
	ext := path.Ext(p)
	base := strings.TrimSuffix(p, ext)
	return base + "." + h + ext
}

var cssURLRe = regexp.MustCompile(`url\(\s*(['"]?)([^)'"]+?)\s*\)`)

// rewriteCSSURLs rewrites relative url() references in CSS to their
// fingerprinted manifest URLs when present (FRK-AS-033).
func rewriteCSSURLs(cssStr string, manifest map[string]string) string {
	return cssURLRe.ReplaceAllStringFunc(cssStr, func(m string) string {
		sm := cssURLRe.FindStringSubmatch(m)
		if len(sm) < 3 {
			return m
		}
		ref := sm[2]
		if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") || strings.HasPrefix(ref, "//") || strings.HasPrefix(ref, "data:") {
			return m
		}
		clean := strings.TrimPrefix(ref, "/")
		if u, ok := manifest[clean]; ok {
			return "url(" + sm[1] + u + sm[1] + ")"
		}
		return m
	})
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func swapExt(p, ext string) string {
	return strings.TrimSuffix(p, path.Ext(p)) + ext
}

// assetURL maps a logical asset path to its served URL. Source asset names may
// include or omit the "assets/" prefix; both resolve under /assets/.
func assetURL(name string) string {
	name = strings.TrimPrefix(name, "/")
	name = strings.TrimPrefix(name, "assets/")
	return "/assets/" + name
}
