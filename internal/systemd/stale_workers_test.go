package systemd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func stageUnits(t *testing.T, names ...string) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	dir := config.SystemdUserDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("[Service]\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", n, err)
		}
	}
}

func unitList(units []WorkerUnit) []string {
	out := make([]string, len(units))
	for i, u := range units {
		out[i] = u.Unit
	}
	return out
}

// The unit a retired worker leaves behind is usually not running at all: it
// failed, or it was never started this boot. FindOrphanedWorkers only answers
// for the running ones, which is exactly why it never saw this case.
func TestStaleWorkerUnits_includesUnitsThatAreNotRunning(t *testing.T) {
	stageUnits(t, "lerd-jump-shop.service", "lerd-queue-shop.service", "lerd-nginx.service")

	got := unitList(StaleWorkerUnits("shop", map[string]bool{"queue": true}))
	if len(got) != 1 || got[0] != "lerd-jump-shop" {
		t.Errorf("got %v, want [lerd-jump-shop]", got)
	}
}

// lerd-queue-admin-shop is admin-shop's queue, not shop's "queue-admin", and
// deleting it as shop's would take a worker off a site that still declares it.
func TestStaleWorkerUnits_skipsALongerSiteCollision(t *testing.T) {
	stageUnits(t, "lerd-queue-admin-shop.service")
	for _, s := range []config.Site{
		{Name: "shop", Path: "/p/shop", Domains: []string{"shop.test"}},
		{Name: "admin-shop", Path: "/p/admin-shop", Domains: []string{"admin-shop.test"}},
	} {
		if err := config.AddSite(s); err != nil {
			t.Fatalf("AddSite: %v", err)
		}
	}

	if got := unitList(StaleWorkerUnits("shop", map[string]bool{})); len(got) != 0 {
		t.Errorf("got %v, want nothing", got)
	}
}
