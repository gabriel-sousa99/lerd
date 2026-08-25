package config

import (
	"strings"
	"testing"
)

const laravelWithWorkers = `name: laravel
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

// A worker that declares no colour of its own is drawn in its framework's, so
// two schedulers from different products never read as the same thing.
func TestWorkerMarks_AWorkerTakesItsFrameworkColour(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	writeCachedFramework(t, "laravel@12.yaml", laravelWithWorkers)

	got := WorkerMarks()
	queue, ok := got.Workers["laravel/queue"]
	if !ok {
		t.Fatalf("laravel/queue missing: %v", got.Workers)
	}
	if queue.Icon != "queue" {
		t.Errorf("icon = %q, want the declared glyph", queue.Icon)
	}
	if queue.Color != "#ff2d20" {
		t.Errorf("colour = %q, want the framework's normalized hex", queue.Color)
	}
}

// A mark with a tone of its own keeps it: Vite inked Laravel red would read as
// a different product.
func TestWorkerMarks_ADeclaredColourWins(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	writeCachedFramework(t, "laravel@12.yaml", laravelWithWorkers)

	vite := WorkerMarks().Workers["laravel/vite"]
	if vite.Color != "#9135ff" {
		t.Errorf("colour = %q, want the worker's own", vite.Color)
	}
}

// The drawings are keyed by icon name rather than by worker, so every framework
// that runs Vite shares one mark instead of shipping its own copy.
func TestWorkerMarks_ServesTheCachedDrawings(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	writeCachedFramework(t, "laravel@12.yaml", laravelWithWorkers)
	if err := SaveStoreWorkerIcon("vite", []byte(`<svg viewBox="0 0 24 24" fill="#9135ff"><path d="M2 2h8v8H2z"/></svg>`)); err != nil {
		t.Fatalf("SaveStoreWorkerIcon: %v", err)
	}

	marks := WorkerMarks().Marks
	svg, ok := marks["vite"]
	if !ok {
		t.Fatalf("vite mark missing: %v", marks)
	}
	if strings.Contains(svg, "#9135ff") {
		t.Errorf("the mark kept a colour of its own: %s", svg)
	}
	if !strings.Contains(svg, `d="M2 2h8v8H2z"`) {
		t.Errorf("the mark lost its drawing: %s", svg)
	}
}

// A worker that declares nothing has nothing to say about how it is drawn, and
// an entry for it would only make the dashboard ask for a mark that cannot exist.
func TestWorkerMarks_SkipsAWorkerThatDeclaresNothing(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	writeCachedFramework(t, "tempest@1.yaml", "name: tempest\nlabel: Tempest\nversion: \"1\"\nworkers:\n  schedule:\n    command: ./tempest schedule:run\n")

	if got := WorkerMarks().Workers; len(got) != 0 {
		t.Errorf("want no entries, got %v", got)
	}
}

// Every version of a framework carries the same worker block, so the versions
// must collapse to one entry rather than fighting over the key.
func TestWorkerMarks_VersionsCollapseToOneEntry(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	writeCachedFramework(t, "laravel@11.yaml", laravelWithWorkers)
	writeCachedFramework(t, "laravel@12.yaml", laravelWithWorkers)

	if got := WorkerMarks().Workers; len(got) != 2 {
		t.Errorf("want queue and vite once each, got %v", got)
	}
}

// Versions are numbers, not strings: a machine that once resolved a Laravel 9
// project still has laravel@9.yaml beside laravel@12.yaml, and sorting the
// names would let 9 outrank 12 and answer with a definition two majors old.
func TestWorkerMarks_TheNewestVersionAnswers(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	writeCachedFramework(t, "laravel@9.yaml", `name: laravel
label: Laravel
version: "9"
workers:
  queue:
    label: Queue Worker
    command: php artisan queue:work
    icon: queue
`)
	writeCachedFramework(t, "laravel@12.yaml", laravelWithWorkers)

	got := WorkerMarks()
	if _, ok := got.Workers["laravel/vite"]; !ok {
		t.Errorf("the newer definition's worker is missing, an older file answered: %v", got.Workers)
	}
}
