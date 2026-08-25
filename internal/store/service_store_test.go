package store

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func serviceTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"services":[
			{"name":"demo","description":"Demo document store","family":"demo","dashboard":"http://localhost:9","depends_on":["mysql"],"color":"#00758F"},
			{"name":"mariadb","description":"MariaDB","family":"mariadb","env_role":"mysql"},
			{"name":"widget","description":"Widget cache"}
		]}`))
	})
	mux.HandleFunc("/demo.yaml", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("name: demo\nimage: example/demo:1\ndescription: Demo document store\ncolor: \"#00758F\"\nports:\n  - \"1234:1234\"\n"))
	})
	mux.HandleFunc("/demo.svg", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#00758f"><path d="M3 3h18v18H3z"/></svg>`))
	})
	mux.HandleFunc("/bad.yaml", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("name: bad\n")) // no image and no versions => invalid
	})
	return httptest.NewServer(mux)
}

func serviceTestClient(srv *httptest.Server) *Client {
	return &Client{BaseURL: srv.URL}
}

func TestFetchServiceIndex(t *testing.T) {
	srv := serviceTestServer(t)
	defer srv.Close()
	idx, err := serviceTestClient(srv).FetchServiceIndex()
	if err != nil {
		t.Fatalf("FetchServiceIndex: %v", err)
	}
	if len(idx.Services) != 3 || idx.Services[0].Name != "demo" {
		t.Fatalf("unexpected index: %+v", idx)
	}
	if idx.Services[0].Family != "demo" || idx.Services[0].Dashboard == "" {
		t.Errorf("index entry lost metadata: %+v", idx.Services[0])
	}
	if idx.Services[1].EnvRole != "mysql" {
		t.Errorf("index entry lost env_role: %+v", idx.Services[1])
	}
	if idx.Services[0].Color != "#00758F" {
		t.Errorf("index entry lost color: %+v", idx.Services[0])
	}
}

// The mark travels with the definition: fetching a preset caches its icon
// beside the YAML, sanitized, so the dashboard has it offline.
func TestFetchServicePreset_CachesTheIconBesideTheYAML(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	srv := serviceTestServer(t)
	defer srv.Close()

	if _, err := serviceTestClient(srv).FetchServicePreset("demo"); err != nil {
		t.Fatalf("FetchServicePreset: %v", err)
	}
	svg, ok := config.PresetIcon("demo")
	if !ok {
		t.Fatal("the preset's icon was not cached")
	}
	if !strings.Contains(svg, `d="M3 3h18v18H3z"`) {
		t.Errorf("cached icon lost its drawing: %s", svg)
	}
	if strings.Contains(svg, "#00758f") {
		t.Errorf("cached icon kept a colour of its own: %s", svg)
	}
	p, err := config.LoadPreset("demo")
	if err != nil || config.NormalizeBrandColor(p.Color) != "#00758f" {
		t.Errorf("preset colour = %q, %v", p.Color, err)
	}
}

// A preset with no icon in the store is the common case, so a missing file must
// leave the fetch itself successful.
func TestFetchServicePreset_SucceedsWithoutAnIcon(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	srv := serviceTestServer(t)
	defer srv.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/plain.yaml", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("name: plain\nimage: example/plain:1\n"))
	})
	iconless := httptest.NewServer(mux)
	defer iconless.Close()

	if _, err := serviceTestClient(iconless).FetchServicePreset("plain"); err != nil {
		t.Fatalf("FetchServicePreset without an icon: %v", err)
	}
	if _, ok := config.PresetIcon("plain"); ok {
		t.Error("no icon in the store should mean no icon in the cache")
	}
}

// The discovery grid shows presets nobody here has installed, so their marks
// have to be cached from the index rather than only on install.
func TestRefreshServiceIcons_CachesMarksForPresetsThatAreNotInstalled(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	srv := serviceTestServer(t)
	defer srv.Close()
	c := serviceTestClient(srv)

	if added := c.RefreshServiceIcons(); added != 1 {
		t.Fatalf("first sweep added %d marks, want 1", added)
	}
	if _, ok := config.PresetIcon("demo"); !ok {
		t.Error("demo's mark was not cached")
	}
	// The preset itself is untouched: a mark is not a reason to pull a definition.
	if config.PresetExists("demo") {
		t.Error("sweeping marks must not install the preset")
	}
	// mariadb and widget publish no mark, and demo's is already cached, so a
	// repeat sweep adds nothing.
	if added := c.RefreshServiceIcons(); added != 0 {
		t.Errorf("repeat sweep added %d marks, want 0", added)
	}
}

// An unreachable store is the offline case: the sweep is decoration and must
// never be the thing that fails.
func TestRefreshServiceIcons_SurvivesAnUnreachableStore(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if added := (&Client{BaseURL: "http://127.0.0.1:1"}).RefreshServiceIcons(); added != 0 {
		t.Errorf("unreachable store added %d marks", added)
	}
}

func TestFetchServicePreset_SavesToCacheAndSeamServesIt(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	srv := serviceTestServer(t)
	defer srv.Close()

	data, err := serviceTestClient(srv).FetchServicePreset("demo")
	if err != nil {
		t.Fatalf("FetchServicePreset: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("FetchServicePreset returned no bytes")
	}
	if _, err := os.Stat(filepath.Join(config.StorePresetsDir(), "demo.yaml")); err != nil {
		t.Errorf("preset not written to store cache: %v", err)
	}
	// The seam must now serve the fetched preset by name.
	if !config.PresetExists("demo") {
		t.Error("PresetExists(demo) false after fetch")
	}
	p, err := config.LoadPreset("demo")
	if err != nil || p.Image != "example/demo:1" {
		t.Errorf("LoadPreset(demo) = %+v, %v", p, err)
	}
}

func TestFetchServicePreset_RejectsInvalid(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	srv := serviceTestServer(t)
	defer srv.Close()

	if _, err := serviceTestClient(srv).FetchServicePreset("bad"); err == nil {
		t.Fatal("expected FetchServicePreset to reject an invalid preset")
	}
	if _, err := os.Stat(filepath.Join(config.StorePresetsDir(), "bad.yaml")); err == nil {
		t.Error("invalid preset must not be written to the store cache")
	}
}

// Full production path: the LERD_SERVICES_BASE_URL override flows through
// origin into NewServiceClient, which fetches, validates, saves to the cache
// dir, and the config seam then serves the preset by name.
func TestNewServiceClient_HonorsEnvOverrideEndToEnd(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	srv := serviceTestServer(t)
	defer srv.Close()
	t.Setenv("LERD_SERVICES_BASE_URL", srv.URL)

	if _, err := NewServiceClient().FetchServicePreset("demo"); err != nil {
		t.Fatalf("FetchServicePreset via env override: %v", err)
	}
	if !config.PresetExists("demo") {
		t.Error("seam does not serve the preset fetched through the real client")
	}
}

func TestSearchServices(t *testing.T) {
	srv := serviceTestServer(t)
	defer srv.Close()
	c := serviceTestClient(srv)

	got, err := c.SearchServices("cache") // matches "Widget cache" description
	if err != nil {
		t.Fatalf("SearchServices: %v", err)
	}
	if len(got) != 1 || got[0].Name != "widget" {
		t.Errorf("SearchServices(cache) = %+v, want [widget]", got)
	}
	if all, _ := c.SearchServices(""); len(all) != 3 {
		t.Errorf("empty query should match all, got %d", len(all))
	}
}
