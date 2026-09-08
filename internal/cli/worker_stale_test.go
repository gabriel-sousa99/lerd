package cli

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// stageStaleUnits registers a site whose .lerd.yaml declares one custom worker,
// and writes the given unit files into the sandboxed systemd user directory.
func stageStaleUnits(t *testing.T, units ...string) config.Site {
	t.Helper()
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	projectDir := t.TempDir()
	registerSite(t, "shop", projectDir)

	yaml := "framework: \"\"\ncustom_workers:\n  mailer:\n    command: php mail\n"
	if err := os.WriteFile(filepath.Join(projectDir, ".lerd.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	unitDir := config.SystemdUserDir()
	if err := os.MkdirAll(unitDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, u := range units {
		if err := os.WriteFile(filepath.Join(unitDir, u), []byte("[Service]\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	site, err := config.FindSite("shop")
	if err != nil {
		t.Fatalf("FindSite: %v", err)
	}
	return *site
}

// The unit a retired worker left behind is the only one that goes; a worker the
// project still declares keeps its unit, because removing that one takes a
// running worker off a site that asked for it.
func TestRemoveStaleWorkerUnits_removesOnlyTheUndeclared(t *testing.T) {
	site := stageStaleUnits(t, "lerd-mailer-shop.service", "lerd-jump-shop.service")
	fake := &stopTrackingMgr{}
	swapMgr(t, fake)

	n, err := RemoveStaleWorkerUnits(site)
	if err != nil {
		t.Fatalf("RemoveStaleWorkerUnits: %v", err)
	}
	if n != 1 {
		t.Errorf("removed = %d, want 1", n)
	}
	got := append([]string(nil), fake.removeServiceCalls...)
	sort.Strings(got)
	if len(got) != 1 || got[0] != "lerd-jump-shop" {
		t.Errorf("removeServiceCalls = %v, want [lerd-jump-shop]", got)
	}
}

// An install that cannot resolve what its site declares must delete nothing:
// treating that silence as "nothing is declared" would take every worker unit
// on the machine with it.
func TestRemoveStaleWorkerUnits_removesNothingWithoutADefinition(t *testing.T) {
	cfgHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfgHome)
	projectDir := t.TempDir()
	registerSite(t, "shop", projectDir)
	unitDir := config.SystemdUserDir()
	if err := os.MkdirAll(unitDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unitDir, "lerd-queue-shop.service"), []byte("[Service]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	site, err := config.FindSite("shop")
	if err != nil {
		t.Fatalf("FindSite: %v", err)
	}
	fake := &stopTrackingMgr{}
	swapMgr(t, fake)

	n, err := RemoveStaleWorkerUnits(*site)
	if err != nil || n != 0 || len(fake.removeServiceCalls) != 0 {
		t.Errorf("removed = %d (err=%v), removes=%v, want nothing touched", n, err, fake.removeServiceCalls)
	}
}
