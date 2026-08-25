package docs

import (
	"strings"
	"testing"
)

func page(section, slug, content string) Page {
	return Page{Title: "T", Section: section, Slug: slug, Path: "docs/" + slug + ".md", content: content}
}

func TestPageHref(t *testing.T) {
	from := page("usage", "sites", "")

	tests := []struct {
		dest string
		want string
	}{
		{"../features/mcp.md", "#docs/features/mcp"},
		{"site-groups.md", "#docs/usage/site-groups"},
		{"./service-presets.md", "#docs/usage/service-presets"},
		{"/features/web-ui", "#docs/features/web-ui"},
		{"/configuration#per-project-config", "#docs/configuration#per-project-config"},
		{"../features/git-worktrees.md#env-overrides", "#docs/features/git-worktrees#env-overrides"},
		{"#env-overrides", "#docs/usage/sites#env-overrides"},
		{"/assets/screenshots/dashboard.png", "/docs/assets/screenshots/dashboard.png"},
	}

	for _, tt := range tests {
		if got := PageHref(tt.dest, from); got != tt.want {
			t.Errorf("PageHref(%q) = %q, want %q", tt.dest, got, tt.want)
		}
	}
}

func TestPageHrefFromTopLevelPage(t *testing.T) {
	from := page("", "configuration", "")
	if got, want := PageHref("./features/git-worktrees.md", from), "#docs/features/git-worktrees"; got != want {
		t.Errorf("PageHref() = %q, want %q", got, want)
	}
	if got, want := PageHref("#env-overrides", from), "#docs/configuration#env-overrides"; got != want {
		t.Errorf("PageHref() = %q, want %q", got, want)
	}
}

func TestAssetURL(t *testing.T) {
	tests := []struct {
		dest    string
		section string
		want    string
	}{
		{"/assets/screenshots/dashboard.png", "usage", "/docs/assets/screenshots/dashboard.png"},
		{"images/local.png", "features", "/docs/features/images/local.png"},
		{"https://lerd.sh/a.png", "usage", "https://lerd.sh/a.png"},
	}
	for _, tt := range tests {
		if got := AssetURL(tt.dest, tt.section); got != tt.want {
			t.Errorf("AssetURL(%q, %q) = %q, want %q", tt.dest, tt.section, got, tt.want)
		}
	}
}

func TestRenderHTMLRewritesLinksAndImages(t *testing.T) {
	p := page("usage", "sites", "# Sites\n\nSee [MCP](../features/mcp.md) and [lerd.sh](https://lerd.sh).\n\n![shot](/assets/screenshots/dashboard.png)\n")

	got, err := RenderHTML(p)
	if err != nil {
		t.Fatalf("RenderHTML() error: %v", err)
	}

	for _, want := range []string{
		`href="#docs/features/mcp"`,
		`href="https://lerd.sh"`,
		`target="_blank"`,
		`src="/docs/assets/screenshots/dashboard.png"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderHTML() = %q, want it to contain %q", got, want)
		}
	}
}

func TestRenderHTMLTablesAndHeadingIDs(t *testing.T) {
	p := page("", "configuration", "## Per-project config\n\n| Key | Meaning |\n|---|---|\n| `php` | version |\n")

	got, err := RenderHTML(p)
	if err != nil {
		t.Fatalf("RenderHTML() error: %v", err)
	}
	for _, want := range []string{"<table>", `id="per-project-config"`, "<code>php</code>"} {
		if !strings.Contains(got, want) {
			t.Errorf("RenderHTML() = %q, want it to contain %q", got, want)
		}
	}
}

func TestRenderHTMLNormalizesContainers(t *testing.T) {
	p := page("usage", "services", "::: warning Known limitation\nOnly loopback.\n:::\n")

	got, err := RenderHTML(p)
	if err != nil {
		t.Fatalf("RenderHTML() error: %v", err)
	}
	if !strings.Contains(got, "<blockquote>") || !strings.Contains(got, "Warning: Known limitation") {
		t.Errorf("RenderHTML() = %q, want a blockquote callout", got)
	}
	if strings.Contains(got, ":::") {
		t.Errorf("RenderHTML() leaked container syntax: %q", got)
	}
}

func TestRenderHTMLDropsRawHTML(t *testing.T) {
	p := page("usage", "sites", "<script>alert(1)</script>\n\nSafe text.\n")

	got, err := RenderHTML(p)
	if err != nil {
		t.Fatalf("RenderHTML() error: %v", err)
	}
	if strings.Contains(got, "<script>") {
		t.Errorf("RenderHTML() emitted raw HTML: %q", got)
	}
}

func TestRenderHTMLEveryEmbeddedPage(t *testing.T) {
	for _, p := range BuildRegistry() {
		out, err := RenderHTML(p)
		if err != nil {
			t.Errorf("RenderHTML(%s) error: %v", p.Path, err)
			continue
		}
		if strings.TrimSpace(out) == "" {
			t.Errorf("RenderHTML(%s) produced nothing", p.Path)
		}
		if strings.Contains(out, ":::") {
			t.Errorf("RenderHTML(%s) leaked container syntax", p.Path)
		}
	}
}
