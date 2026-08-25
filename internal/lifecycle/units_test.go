package lifecycle

import (
	"slices"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/podman"
)

// TestQuitProcessUnits_SkipMatchesTheWatcherConstant guards the skip list
// against drift: the watcher passes podman.WatcherUnit, so a rename there must
// not silently stop matching the name QuitProcessUnits returns.
func TestQuitProcessUnits_SkipMatchesTheWatcherConstant(t *testing.T) {
	if slices.Contains(QuitProcessUnits(podman.WatcherUnit), podman.WatcherUnit) {
		t.Errorf("QuitProcessUnits must drop %s when it is skipped", podman.WatcherUnit)
	}
}

// TestQuitProcessUnits_SkipsSelfUnit pins that a caller can drop its own unit
// from the teardown. The watcher quits lerd from inside lerd-watcher: stopping
// that unit would boot out the job it is running in, and the bootout blocks
// until the process exits, so the teardown would never reach the Podman
// Machine stop that the whole shutdown path exists to perform.
func TestQuitProcessUnits_SkipsSelfUnit(t *testing.T) {
	units := QuitProcessUnits("lerd-watcher")
	if slices.Contains(units, "lerd-watcher") {
		t.Error("QuitProcessUnits must drop a skipped unit")
	}
	if !slices.Contains(units, "lerd-dns") {
		t.Error("QuitProcessUnits must still stop the units that were not skipped")
	}
}

// TestQuitProcessUnits_DefaultKeepsEverything pins that the CLI path, which
// passes no skip list, still tears down every process unit.
func TestQuitProcessUnits_DefaultKeepsEverything(t *testing.T) {
	units := QuitProcessUnits()
	for _, want := range []string{"lerd-ui", "lerd-watcher", "lerd-tray", "lerd-dns"} {
		if !slices.Contains(units, want) {
			t.Errorf("QuitProcessUnits() missing %s", want)
		}
	}
}
