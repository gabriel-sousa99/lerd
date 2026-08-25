package docs

import (
	"strings"
	"testing"
)

func TestBuildRegistry(t *testing.T) {
	pages := BuildRegistry()
	if len(pages) == 0 {
		t.Fatal("BuildRegistry() returned no pages")
	}

	byRoute := make(map[string]Page, len(pages))
	for _, p := range pages {
		if p.Title == "" {
			t.Errorf("page %s has no title", p.Path)
		}
		if strings.TrimSpace(p.Content()) == "" {
			t.Errorf("page %s has no content", p.Path)
		}
		if _, dup := byRoute[p.Route()]; dup {
			t.Errorf("duplicate route %s", p.Route())
		}
		byRoute[p.Route()] = p
	}

	for _, route := range []string{"usage/sites", "features/web-ui", "configuration"} {
		if _, ok := byRoute[route]; !ok {
			t.Errorf("registry is missing %s", route)
		}
	}
	// The landing page is frontmatter only and belongs on neither surface.
	if _, ok := byRoute["index"]; ok {
		t.Error("registry includes the frontmatter-only landing page")
	}
}

func TestBuildRegistryOrdersSections(t *testing.T) {
	var order []string
	for _, p := range BuildRegistry() {
		if len(order) == 0 || order[len(order)-1] != p.Section {
			order = append(order, p.Section)
		}
	}

	seen := make(map[string]bool)
	for _, s := range order {
		if seen[s] {
			t.Errorf("section %q is split across the registry: %v", s, order)
		}
		seen[s] = true
	}
	if order[0] != "" || order[1] != "getting-started" {
		t.Errorf("unexpected section order: %v", order)
	}
}

func TestSectionLabel(t *testing.T) {
	if got, want := SectionLabel("getting-started"), "Getting Started"; got != want {
		t.Errorf("SectionLabel() = %q, want %q", got, want)
	}
	if got, want := SectionLabel("brand-new"), "Brand New"; got != want {
		t.Errorf("SectionLabel() = %q, want %q", got, want)
	}
}

func TestFilterPages(t *testing.T) {
	pages := []Page{
		page("usage", "sites", "Linking a project."),
		page("usage", "database", "Snapshots and imports."),
	}
	pages[0].Title = "Site Management"
	pages[1].Title = "Database"

	if got := FilterPages(pages, ""); len(got) != 2 {
		t.Errorf("FilterPages(\"\") returned %d pages, want 2", len(got))
	}
	if got := FilterPages(pages, "snapshot"); len(got) != 1 || got[0].Slug != "database" {
		t.Errorf("FilterPages(\"snapshot\") = %v, want the database page", got)
	}
	if got := FilterPages(pages, "management"); len(got) != 1 || got[0].Slug != "sites" {
		t.Errorf("FilterPages(\"management\") = %v, want the sites page", got)
	}
}

func TestSearch(t *testing.T) {
	sites := page("usage", "sites", "# Sites\n\nLinking a project creates its database too.\n")
	sites.Title = "Site Management"
	db := page("usage", "database", "# Database\n\nSnapshots restore a database in place.\n")
	db.Title = "Database"
	pages := []Page{sites, db}

	if got := Search(pages, "", 10); got != nil {
		t.Errorf("Search(\"\") = %v, want nothing", got)
	}

	got := Search(pages, "snapshots", 10)
	if len(got) != 1 {
		t.Fatalf("Search(\"snapshots\") returned %d results, want 1", len(got))
	}
	if got[0].Route != "usage/database" {
		t.Errorf("result route = %q, want usage/database", got[0].Route)
	}
	if !strings.Contains(got[0].Snippet, "Snapshots restore a database") {
		t.Errorf("snippet = %q, want the matching sentence", got[0].Snippet)
	}

	// A title match outranks a body match for the same word.
	ranked := Search(pages, "database", 10)
	if len(ranked) != 2 || ranked[0].Route != "usage/database" {
		t.Errorf("Search(\"database\") = %v, want the title match first", ranked)
	}

	if got := Search(pages, "a", 1); len(got) != 1 {
		t.Errorf("Search() ignored the limit: got %d results", len(got))
	}
}
