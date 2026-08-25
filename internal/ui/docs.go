package ui

import (
	"encoding/json"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"

	docsfs "github.com/gabriel-sousa99/lerd"
	"github.com/gabriel-sousa99/lerd/internal/docs"
)

// The dashboard reads the documentation out of the same embedded pages `lerd man`
// uses, so a machine with no internet still has it. The routes live outside /api
// on purpose: the service worker caches everything else under the origin, so the
// pages it has already seen survive the daemon going away too.

var docsRegistry = sync.OnceValue(docs.BuildRegistry)

var docsHTML struct {
	sync.Mutex
	byRoute map[string]string
}

// docsPageJSON is one page of the documentation, rendered for the dashboard.
type docsPageJSON struct {
	Title        string `json:"title"`
	Section      string `json:"section"`
	SectionLabel string `json:"section_label"`
	Slug         string `json:"slug"`
	Route        string `json:"route"`
	HTML         string `json:"html,omitempty"`
}

func docsMeta(p docs.Page) docsPageJSON {
	return docsPageJSON{
		Title:        p.Title,
		Section:      p.Section,
		SectionLabel: docs.SectionLabel(p.Section),
		Slug:         p.Slug,
		Route:        p.Route(),
	}
}

func handleDocsIndex(w http.ResponseWriter, _ *http.Request) {
	pages := docsRegistry()
	out := make([]docsPageJSON, 0, len(pages))
	for _, p := range pages {
		out = append(out, docsMeta(p))
	}
	writeDocsJSON(w, map[string]any{"pages": out})
}

func handleDocsPage(w http.ResponseWriter, r *http.Request) {
	route := strings.Trim(strings.TrimPrefix(r.URL.Path, "/docs/page/"), "/")
	for _, p := range docsRegistry() {
		if p.Route() != route {
			continue
		}
		html, err := renderDocsPage(p)
		if err != nil {
			http.Error(w, "rendering "+route+": "+err.Error(), http.StatusInternalServerError)
			return
		}
		out := docsMeta(p)
		out.HTML = html
		writeDocsJSON(w, out)
		return
	}
	http.Error(w, "no documentation page at "+route, http.StatusNotFound)
}

func handleDocsSearch(w http.ResponseWriter, r *http.Request) {
	results := docs.Search(docsRegistry(), r.URL.Query().Get("q"), 30)
	if results == nil {
		results = []docs.Result{}
	}
	writeDocsJSON(w, map[string]any{"results": results})
}

// handleDocsAsset serves the images the pages reference. VitePress resolves a
// site-absolute path against docs/public, so that is tried first, then the docs
// tree itself for the files only the embedded copy carries.
func handleDocsAsset(w http.ResponseWriter, r *http.Request) {
	rel := strings.TrimPrefix(path.Clean("/"+strings.TrimPrefix(r.URL.Path, "/docs/")), "/")
	if rel == "" || rel == "." {
		http.NotFound(w, r)
		return
	}
	for _, base := range []string{"docs/public/", "docs/"} {
		body, err := docsfs.FS.ReadFile(base + rel)
		if err != nil {
			continue
		}
		if ct := mime.TypeByExtension(path.Ext(rel)); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(body) //nolint:errcheck
		return
	}
	http.NotFound(w, r)
}

// renderDocsPage renders a page once and keeps the HTML; the pages are baked
// into the binary, so the result can never go stale while it runs.
func renderDocsPage(p docs.Page) (string, error) {
	docsHTML.Lock()
	defer docsHTML.Unlock()
	if html, ok := docsHTML.byRoute[p.Route()]; ok {
		return html, nil
	}
	html, err := docs.RenderHTML(p)
	if err != nil {
		return "", err
	}
	if docsHTML.byRoute == nil {
		docsHTML.byRoute = make(map[string]string)
	}
	docsHTML.byRoute[p.Route()] = html
	return html, nil
}

func writeDocsJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
