package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func isolateRunDir(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, ".local", "share"))
}

// TestConsumeWatcherManagedStop_TrueOnceThenFalse pins that the marker is a
// one-shot. It has to survive from the write until the watcher reads it, and it
// has to be gone straight after, or the next real logout inherits it and the
// machine powers off with the Podman Machine VM still running.
func TestConsumeWatcherManagedStop_TrueOnceThenFalse(t *testing.T) {
	isolateRunDir(t)

	if err := MarkWatcherManagedStop(); err != nil {
		t.Fatalf("MarkWatcherManagedStop: %v", err)
	}
	if !ConsumeWatcherManagedStop() {
		t.Fatal("a marked stop must read as lerd-initiated")
	}
	if ConsumeWatcherManagedStop() {
		t.Error("the marker must not survive being consumed")
	}
}

// TestConsumeWatcherManagedStop_UnmarkedIsALogout pins the default: with no
// marker the watcher must treat the signal as a real logout and tear down.
func TestConsumeWatcherManagedStop_UnmarkedIsALogout(t *testing.T) {
	isolateRunDir(t)

	if ConsumeWatcherManagedStop() {
		t.Error("an unmarked stop must read as a logout")
	}
}

// TestConsumeWatcherManagedStop_StaleMarkerIsIgnored pins the TTL. A marker
// whose watcher died before reading it must not suppress a later logout.
func TestConsumeWatcherManagedStop_StaleMarkerIsIgnored(t *testing.T) {
	isolateRunDir(t)

	if err := MarkWatcherManagedStop(); err != nil {
		t.Fatalf("MarkWatcherManagedStop: %v", err)
	}
	stale := time.Now().Add(-watcherManagedStopTTL - time.Minute)
	if err := os.Chtimes(watcherManagedStopMarkerPath(), stale, stale); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	if ConsumeWatcherManagedStop() {
		t.Error("a marker older than the TTL must not suppress a logout teardown")
	}
	if _, err := os.Stat(watcherManagedStopMarkerPath()); !os.IsNotExist(err) {
		t.Error("a stale marker must still be cleared")
	}
}
