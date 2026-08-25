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

func writeWorkerFrameworkDef(t *testing.T) {
	t.Helper()
	dir := config.StoreFrameworksDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `name: laravel
label: Laravel
version: "12"
color: "#FF2D20"
workers:
  queue:
    label: Queue Worker
    command: php artisan queue:work
    icon: queue
  vite:
    label: Vite
    command: npm run dev
    icon: vite
    color: "#9135FF"
`
	if err := os.WriteFile(filepath.Join(dir, "laravel@12.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The dashboard draws worker marks the same way it draws framework ones: out of
// this install's own sanitized cache, never from the store origin.
func TestHandleWorkerMarks_ServesWhatEachWorkerAsksFor(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	writeWorkerFrameworkDef(t)
	raw := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#9135ff"><path d="M4 4h6v6H4z"/><script>bad()</script></svg>`
	if err := config.SaveStoreWorkerIcon("vite", []byte(raw)); err != nil {
		t.Fatalf("SaveStoreWorkerIcon: %v", err)
	}

	rec := httptest.NewRecorder()
	handleWorkerMarks(rec, httptest.NewRequest(http.MethodGet, "/api/workers/marks", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got config.WorkerMarkSet
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Workers["laravel/queue"].Color != "#ff2d20" {
		t.Errorf("queue should take Laravel's colour, got %+v", got.Workers["laravel/queue"])
	}
	if got.Workers["laravel/vite"].Icon != "vite" {
		t.Errorf("vite should name its mark, got %+v", got.Workers["laravel/vite"])
	}
	svg := got.Marks["vite"]
	if strings.Contains(svg, "bad()") || strings.Contains(svg, "#9135ff") {
		t.Errorf("the served mark was not the sanitized copy: %s", svg)
	}
	if !strings.Contains(svg, `d="M4 4h6v6H4z"`) {
		t.Errorf("the served mark lost its drawing: %s", svg)
	}
}

// An install with nothing cached must answer with empty maps rather than nulls,
// so the dashboard can index into them without a guard on every read.
func TestHandleWorkerMarks_EmptyInstallServesEmptyMaps(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	rec := httptest.NewRecorder()
	handleWorkerMarks(rec, httptest.NewRequest(http.MethodGet, "/api/workers/marks", nil))
	if body := strings.TrimSpace(rec.Body.String()); !strings.Contains(body, `"workers":{}`) ||
		!strings.Contains(body, `"marks":{}`) {
		t.Errorf("want empty maps, got %s", body)
	}
}

func TestHandleWorkerMarks_RejectsAPost(t *testing.T) {
	rec := httptest.NewRecorder()
	handleWorkerMarks(rec, httptest.NewRequest(http.MethodPost, "/api/workers/marks", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("want 405, got %d", rec.Code)
	}
}
