package ssg

import (
	"bytes"
	"html/template"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func scopeFragment(name, body string) (template.HTML, error) {
	frag, err := html.ParseFragment(strings.NewReader(body), &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	})
	if err != nil {
		return "", err
	}
	target := firstElement(frag)
	if target == nil {
		return template.HTML(body), nil
	}
	for _, a := range target.Attr {
		if a.Key == "data-kiw-component" {
			return template.HTML(body), nil
		}
	}
	target.Attr = append(target.Attr, html.Attribute{Key: "data-kiw-component", Val: name})
	var buf bytes.Buffer
	for _, n := range frag {
		if err := html.Render(&buf, n); err != nil {
			return "", err
		}
	}
	return template.HTML(buf.String()), nil
}

func scopeDocument(name, doc string) (template.HTML, error) {
	root, err := html.Parse(strings.NewReader(doc))
	if err != nil {
		return "", err
	}
	for n := root.FirstChild; n != nil; n = n.NextSibling {
		if n.Type == html.ElementNode && n.Data == "html" {
			n.Attr = append(n.Attr, html.Attribute{Key: "data-kiw-layout", Val: name})
			break
		}
	}
	var buf bytes.Buffer
	if err := html.Render(&buf, root); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

func firstElement(frag []*html.Node) *html.Node {
	for _, n := range frag {
		if n.Type == html.ElementNode {
			return n
		}
	}
	return nil
}
