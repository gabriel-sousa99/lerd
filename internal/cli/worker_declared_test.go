package cli

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/services"
)

// listUnitsMgr answers the unit-file listing off a fixed set, which is what
// tells a worker the user still wants from one that was stopped.
type listUnitsMgr struct {
	stopTrackingMgr
	units []string
}

func (l *listUnitsMgr) ListServiceUnits(string) []string { return slices.Clone(l.units) }
func (l *listUnitsMgr) ListTimerUnits(string) []string   { return nil }

// useServiceMgr installs m for the duration of the test. The linux-only test
// files have their own swap helper; this one has to build on macOS too.
func useServiceMgr(t *testing.T, m services.ServiceManager) {
	t.Helper()
	prev := services.Mgr
	services.Mgr = m
	t.Cleanup(func() { services.Mgr = prev })
}

// writeWorkerFrameworkFixture puts a store definition carrying the named
// workers where GetFrameworkForDir will resolve it for a site on that framework.
func writeWorkerFrameworkFixture(t *testing.T, sitePath string, workers ...string) {
	t.Helper()
	storeDir := config.StoreFrameworksDir()
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	def := "name: laravel\nlabel: Laravel\nworkers:\n"
	for _, w := range workers {
		def += "  " + w + ":\n    label: " + w + "\n    command: php artisan " + w + "\n"
	}
	if err := os.WriteFile(filepath.Join(storeDir, "laravel.yaml"), []byte(def), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sitePath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sitePath, ".lerd.yaml"), []byte("framework: laravel\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The declared set follows the units on disk rather than what is running. A
// worker that is merely down when another one is touched still has its unit,
// and writing it out of .lerd.yaml is what stopped lerd start from ever
// bringing it back (#1627).
func TestCollectDeclaredWorkerNames_keepsAWorkerThatIsNotRunning(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	sitePath := filepath.Join(t.TempDir(), "app")
	writeWorkerFrameworkFixture(t, sitePath, "queue", "horizon")
	useServiceMgr(t, &listUnitsMgr{units: []string{"lerd-queue-app", "lerd-horizon-app"}})

	got := CollectDeclaredWorkerNames(&config.Site{Name: "app", Path: sitePath, Framework: "laravel"})
	for _, want := range []string{"queue", "horizon"} {
		if !slices.Contains(got, want) {
			t.Errorf("worker %q has a unit installed and must stay declared, got %v", want, got)
		}
	}
}

// Stopping a worker removes its unit, which is how the set learns the user is
// done with it.
func TestCollectDeclaredWorkerNames_dropsAWorkerWithNoUnit(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	sitePath := filepath.Join(t.TempDir(), "app")
	writeWorkerFrameworkFixture(t, sitePath, "queue", "horizon")
	useServiceMgr(t, &listUnitsMgr{units: []string{"lerd-queue-app"}})

	got := CollectDeclaredWorkerNames(&config.Site{Name: "app", Path: sitePath, Framework: "laravel"})
	if slices.Contains(got, "horizon") {
		t.Errorf("a worker whose unit a stop removed must leave the set, got %v", got)
	}
	if !slices.Contains(got, "queue") {
		t.Errorf("the remaining worker must stay declared, got %v", got)
	}
}
