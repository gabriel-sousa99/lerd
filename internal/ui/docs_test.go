package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleDocsIndex(t *testing.T) {
	rec := httptest.NewRecorder()
	handleDocsIndex(rec, httptest.NewRequest(http.MethodGet, "/docs/index.json", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body struct {
		Pages []docsPageJSON `json:"pages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding index: %v", err)
	}
	if len(body.Pages) == 0 {
		t.Fatal("index lists no pages")
	}

	var found bool
	for _, p := range body.Pages {
		if p.Route == "usage/sites" {
			found = true
			if p.SectionLabel != "Usage" || p.Title == "" {
				t.Errorf("unexpected entry for usage/sites: %+v", p)
			}
		}
		if p.HTML != "" {
			t.Errorf("index carries page HTML for %s", p.Route)
		}
	}
	if !found {
		t.Error("index is missing usage/sites")
	}
}

func TestHandleDocsPage(t *testing.T) {
	rec := httptest.NewRecorder()
	handleDocsPage(rec, httptest.NewRequest(http.MethodGet, "/docs/page/usage/sites", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var page docsPageJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decoding page: %v", err)
	}
	if page.Route != "usage/sites" || page.Title == "" {
		t.Errorf("unexpected page: %+v", page)
	}
	if !strings.Contains(page.HTML, "<h1") {
		t.Errorf("page HTML looks unrendered: %q", firstBytes(page.HTML))
	}
	if strings.Contains(page.HTML, "](") || strings.Contains(page.HTML, ":::") {
		t.Errorf("page HTML leaked markdown: %q", firstBytes(page.HTML))
	}
}

func TestHandleDocsPageUnknown(t *testing.T) {
	rec := httptest.NewRecorder()
	handleDocsPage(rec, httptest.NewRequest(http.MethodGet, "/docs/page/usage/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleDocsSearch(t *testing.T) {
	rec := httptest.NewRecorder()
	handleDocsSearch(rec, httptest.NewRequest(http.MethodGet, "/docs/search?q=worktree", nil))

	var body struct {
		Results []struct {
			Route   string `json:"route"`
			Snippet string `json:"snippet"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decoding search: %v", err)
	}
	if len(body.Results) == 0 {
		t.Fatal("search found nothing for a word the docs use")
	}
	if body.Results[0].Route == "" {
		t.Errorf("result has no route: %+v", body.Results[0])
	}

	rec = httptest.NewRecorder()
	handleDocsSearch(rec, httptest.NewRequest(http.MethodGet, "/docs/search?q=", nil))
	if !strings.Contains(rec.Body.String(), `"results":[]`) {
		t.Errorf("blank query returned %q, want an empty list", rec.Body.String())
	}
}

func TestHandleDocsAsset(t *testing.T) {
	rec := httptest.NewRecorder()
	handleDocsAsset(rec, httptest.NewRequest(http.MethodGet, "/docs/assets/screenshots/dashboard.png", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}
	if rec.Body.Len() == 0 {
		t.Error("asset body is empty")
	}
}

func TestHandleDocsAssetRejectsTraversal(t *testing.T) {
	for _, p := range []string{"/docs/../go.mod", "/docs/%2e%2e/go.mod", "/docs/", "/docs/nope.png"} {
		rec := httptest.NewRecorder()
		handleDocsAsset(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404", p, rec.Code)
		}
	}
}

func firstBytes(s string) string {
	if len(s) > 200 {
		return s[:200]
	}
	return s
}
