package config

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Both stores let a definition carry its own mark next to its YAML, cached in
// the same directory. The markup is remote and ends up in an {@html} block, so
// it is cut down to a plain drawing on the way in, the same spirit as
// SanitizeProjectFrameworkDef for untrusted project YAML. What comes out is a
// monochrome silhouette with no colours of its own; the dashboard paints it
// through currentColor with the definition's declared brand colour.
//
// This file is the half both stores share: the sanitiser, the colour rule, and
// the read and write of a cached mark. The per-store halves are in
// preset_icon.go and framework_icon.go.

const (
	maxIconBytes    = 32 * 1024
	maxIconElements = 400
	svgNS           = "http://www.w3.org/2000/svg"
)

// iconElements is the drawing subset an icon may use. Anything else, script and
// foreignObject included, is dropped with its whole subtree.
var iconElements = map[string]bool{
	"g": true, "path": true, "circle": true, "ellipse": true,
	"rect": true, "line": true, "polyline": true, "polygon": true,
}

// iconAttrs is the geometry an element may keep. Every presentation attribute
// is dropped, so the icon cannot carry a colour, reach outside itself, or smuggle
// CSS: no fill, stroke, style, class, id, href or event handler survives.
var iconAttrs = map[string]bool{
	"d": true, "points": true, "transform": true,
	"x": true, "y": true, "x1": true, "y1": true, "x2": true, "y2": true,
	"cx": true, "cy": true, "r": true, "rx": true, "ry": true,
	"width": true, "height": true,
	"fill-rule": true, "clip-rule": true,
}

// SanitizeStoreIcon reduces remote SVG markup to a bare drawing: a single
// <svg> carrying only its viewBox, wrapping the allowed shape elements with
// their geometry attributes. It fails rather than returning something empty, so
// a caller never caches markup that would draw nothing.
func SanitizeStoreIcon(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("empty icon")
	}
	if len(data) > maxIconBytes {
		return nil, fmt.Errorf("icon is %d bytes, over the %d byte limit", len(data), maxIconBytes)
	}

	dec := xml.NewDecoder(strings.NewReader(string(data)))
	var (
		out      strings.Builder
		viewBox  string
		started  bool
		shapes   int
		elements int
		skipDep  int
		open     []string
	)
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parsing icon: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			// Inside a dropped element everything goes with it, however deep.
			if skipDep > 0 {
				skipDep++
				continue
			}
			local := t.Name.Local
			if !started {
				if local != "svg" {
					return nil, fmt.Errorf("icon root is <%s>, not <svg>", local)
				}
				started = true
				viewBox = rootViewBox(t)
				continue
			}
			elements++
			if elements > maxIconElements {
				return nil, fmt.Errorf("icon has over %d elements", maxIconElements)
			}
			if !iconElements[local] || (t.Name.Space != "" && t.Name.Space != svgNS) {
				skipDep = 1
				continue
			}
			if local != "g" {
				shapes++
			}
			out.WriteString("<" + local)
			writeIconAttrs(&out, t.Attr)
			out.WriteString(">")
			open = append(open, local)
		case xml.EndElement:
			if skipDep > 0 {
				skipDep--
				continue
			}
			if len(open) == 0 {
				continue
			}
			out.WriteString("</" + open[len(open)-1] + ">")
			open = open[:len(open)-1]
		}
	}

	if !started {
		return nil, errors.New("icon has no <svg> root")
	}
	if viewBox == "" {
		return nil, errors.New("icon declares no viewBox")
	}
	if shapes == 0 {
		return nil, errors.New("icon draws nothing")
	}
	return []byte(`<svg xmlns="` + svgNS + `" viewBox="` + viewBox + `">` + out.String() + `</svg>`), nil
}

// rootViewBox reads the root element's viewBox, falling back to one derived from
// a plain numeric width and height so an icon exported without a viewBox still
// scales instead of being rejected.
func rootViewBox(root xml.StartElement) string {
	var w, h string
	for _, a := range root.Attr {
		switch a.Name.Local {
		case "viewBox":
			if v := numericList(a.Value); v != "" {
				return v
			}
		case "width":
			w = a.Value
		case "height":
			h = a.Value
		}
	}
	if isNumber(w) && isNumber(h) {
		return "0 0 " + w + " " + h
	}
	return ""
}

// writeIconAttrs emits the geometry attributes an element is allowed to keep.
// Namespaced attributes (xlink:href and friends) are dropped wholesale.
func writeIconAttrs(out *strings.Builder, attrs []xml.Attr) {
	for _, a := range attrs {
		if a.Name.Space != "" || !iconAttrs[a.Name.Local] {
			continue
		}
		out.WriteString(" " + a.Name.Local + `="`)
		xml.EscapeText(out, []byte(a.Value))
		out.WriteString(`"`)
	}
}

// numericList keeps a whitespace or comma separated list of numbers, so a
// viewBox can never carry anything but coordinates.
func numericList(v string) string {
	fields := strings.FieldsFunc(v, func(r rune) bool { return r == ' ' || r == ',' || r == '\t' || r == '\n' })
	if len(fields) != 4 {
		return ""
	}
	for _, f := range fields {
		if !isNumber(f) {
			return ""
		}
	}
	return strings.Join(fields, " ")
}

func isNumber(v string) bool {
	if v == "" {
		return false
	}
	_, err := strconv.ParseFloat(v, 64)
	return err == nil
}

// NormalizeBrandColor returns the preset's declared colour as a lowercase hex
// literal, or empty when it is anything else. The value reaches the dashboard as
// a custom property on the element, so only a plain hex may pass: arbitrary CSS
// would be a value out of the store deciding how a page renders.
func NormalizeBrandColor(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if len(v) != 4 && len(v) != 7 {
		return ""
	}
	if !strings.HasPrefix(v, "#") {
		return ""
	}
	for _, c := range v[1:] {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return ""
		}
	}
	return v
}

// saveStoreIcon sanitizes a mark fetched from a store and writes it into that
// store's cache dir. Unlike the YAML beside it the original bytes are not kept:
// what a future binary would otherwise have to trust is exactly what an older
// one already stripped, so the cache holds the safe form.
func saveStoreIcon(dir, name string, data []byte) error {
	if !validPresetName(name) {
		return fmt.Errorf("invalid definition name %q", name)
	}
	clean, err := SanitizeStoreIcon(data)
	if err != nil {
		return fmt.Errorf("icon for %q: %w", name, err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, name+".svg")
	guardRealWrite(path)
	return publishStoreFile(path, clean, 0o644)
}

// readStoreIcon returns the cached mark for a name, if one was ever fetched.
func readStoreIcon(dir, name string) (string, bool) {
	if !validPresetName(name) {
		return "", false
	}
	data, err := os.ReadFile(filepath.Join(dir, name+".svg"))
	if err != nil {
		return "", false
	}
	return string(data), true
}

// removeStoreIcon drops a cached mark, called wherever the cached definition
// goes so the two never outlive each other.
func removeStoreIcon(dir, name string) error {
	if !validPresetName(name) {
		return nil
	}
	err := os.Remove(filepath.Join(dir, name+".svg"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// iconNames lists the names an icon layer can serve.
func iconNames(fsys fs.FS, dir string) []string {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".svg") {
			continue
		}
		if name := strings.TrimSuffix(e.Name(), ".svg"); validPresetName(name) {
			out = append(out, name)
		}
	}
	return out
}
