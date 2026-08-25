package docs

import (
	"bytes"
	"path"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// AssetPrefix is where the dashboard serves the images the docs reference.
const AssetPrefix = "/docs/"

// RoutePrefix is the dashboard hash route a documentation page lives at.
const RoutePrefix = "#docs/"

// RenderHTML renders a page to an HTML fragment for the dashboard. Links between
// pages become dashboard routes and images become URLs the binary serves, so a
// reader never leaves the machine. Raw HTML is dropped rather than emitted.
func RenderHTML(p Page) (string, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(util.Prioritized(&linkRewriter{page: p}, 100)),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert([]byte(Normalize(p.Content())), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type linkRewriter struct {
	page Page
}

func (t *linkRewriter) Transform(doc *ast.Document, _ text.Reader, _ parser.Context) {
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch v := n.(type) {
		case *ast.Image:
			v.Destination = []byte(AssetURL(string(v.Destination), t.page.Section))
		case *ast.Link:
			dest := string(v.Destination)
			if isExternal(dest) {
				v.SetAttributeString("target", []byte("_blank"))
				v.SetAttributeString("rel", []byte("noopener noreferrer"))
				return ast.WalkContinue, nil
			}
			v.Destination = []byte(PageHref(dest, t.page))
		}
		return ast.WalkContinue, nil
	})
}

// PageHref rewrites a link between documentation pages into the dashboard route
// that shows it. Anchors survive, and an image or download link keeps pointing at
// the served file.
func PageHref(dest string, from Page) string {
	if dest == "" {
		return dest
	}
	if strings.HasPrefix(dest, "#") {
		return RoutePrefix + from.Route() + dest
	}

	target, anchor, _ := strings.Cut(dest, "#")
	if anchor != "" {
		anchor = "#" + anchor
	}
	if ext := path.Ext(target); ext != "" && ext != ".md" {
		return AssetURL(dest, from.Section)
	}

	target = strings.TrimSuffix(target, ".md")
	if strings.HasPrefix(target, "/") {
		target = strings.TrimPrefix(target, "/")
	} else {
		target = path.Join(from.Section, target)
	}
	target = strings.TrimSuffix(path.Clean("/"+target), "/")
	return RoutePrefix + strings.TrimPrefix(target, "/") + anchor
}

// AssetURL rewrites a file reference in the docs into the URL the binary serves
// it from. Site-absolute paths keep their shape under the prefix, relative ones
// resolve against the page's section.
func AssetURL(dest, section string) string {
	if dest == "" || isExternal(dest) || strings.HasPrefix(dest, "data:") {
		return dest
	}
	if !strings.HasPrefix(dest, "/") {
		dest = "/" + path.Join(section, dest)
	}
	return strings.TrimSuffix(AssetPrefix, "/") + path.Clean(dest)
}

func isExternal(dest string) bool {
	return strings.Contains(dest, "://") || strings.HasPrefix(dest, "mailto:")
}
