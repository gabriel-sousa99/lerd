package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// The dashboard inlines these marks, so the endpoint must hand it the sanitized
// cached copy and nothing the browser has to fetch from the store itself.
func TestHandleFrameworkMarks_ServesTheCachedMarkAndColour(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	dir := config.StoreFrameworksDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := "name: laravel\nlabel: Laravel\nversion: \"12\"\ncolor: \"#FF2D20\"\n"
	if err := os.WriteFile(filepath.Join(dir, "laravel@12.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	raw := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#f00"><path d="M4 4h6v6H4z"/><script>bad()</script></svg>`
	if err := config.SaveStoreFrameworkIcon("laravel", []byte(raw)); err != nil {
		t.Fatalf("SaveStoreFrameworkIcon: %v", err)
	}

	rec := httptest.NewRecorder()
	handleFrameworkMarks(rec, httptest.NewRequest(http.MethodGet, "/api/frameworks/marks", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got map[string]config.FrameworkMark
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	mark, ok := got["laravel"]
	if !ok {
		t.Fatalf("marks response missing laravel: %v", got)
	}
	if !strings.Contains(mark.SVG, `d="M4 4h6v6H4z"`) {
		t.Errorf("served mark lost its drawing: %s", mark.SVG)
	}
	if strings.Contains(mark.SVG, "bad()") || strings.Contains(mark.SVG, "#f00") {
		t.Errorf("served mark is not the sanitized copy: %s", mark.SVG)
	}
	if mark.Color != "#ff2d20" {
		t.Errorf("served colour = %q", mark.Color)
	}
}

// With nothing cached the dashboard must get an object it can iterate, not a
// null it has to guard against.
func TestHandleFrameworkMarks_EmptyCacheServesAnObject(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	rec := httptest.NewRecorder()
	handleFrameworkMarks(rec, httptest.NewRequest(http.MethodGet, "/api/frameworks/marks", nil))
	if body := strings.TrimSpace(rec.Body.String()); body != "{}" {
		t.Errorf("empty cache should serve an empty object, got %q", body)
	}
}
