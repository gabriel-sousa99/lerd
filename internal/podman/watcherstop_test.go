package podman

import (
	"path/filepath"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func isolateRunDir(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, ".local", "share"))
}

// TestStopUnit_MarksTheWatcher covers the platform-independent funnel. On Linux
// the service manager delegates straight to StopUnit, so this is where a
// systemd stop of the watcher gets marked as lerd-initiated; without it a
// `lerd install` on Linux would tear the environment down mid-install.
func TestStopUnit_MarksTheWatcher(t *testing.T) {
	isolateRunDir(t)

	_ = StopUnit(WatcherUnit)
	if !config.ConsumeWatcherManagedStop() {
		t.Error("StopUnit must mark a watcher stop as lerd-initiated")
	}

	_ = StopUnit("lerd-nginx")
	if config.ConsumeWatcherManagedStop() {
		t.Error("StopUnit must not mark a watcher stop for any other unit")
	}
}

// TestRestartUnit_MarksTheWatcher covers the same funnel for restarts, which is
// the path `lerd install` takes when it refreshes the watcher service.
func TestRestartUnit_MarksTheWatcher(t *testing.T) {
	isolateRunDir(t)

	_ = RestartUnit(WatcherUnit)
	if !config.ConsumeWatcherManagedStop() {
		t.Error("RestartUnit must mark a watcher stop as lerd-initiated")
	}
}

// TestMarkManagedWatcherStop_OnlyTheWatcher pins the name check itself, so a
// future funnel calling it with an arbitrary unit cannot suppress a real logout.
func TestMarkManagedWatcherStop_OnlyTheWatcher(t *testing.T) {
	isolateRunDir(t)

	for _, unit := range []string{"lerd-ui", "lerd-dns", "lerd-watcher-thing", ""} {
		MarkManagedWatcherStop(unit)
		if config.ConsumeWatcherManagedStop() {
			t.Errorf("MarkManagedWatcherStop(%q) must not mark a watcher stop", unit)
		}
	}
}
