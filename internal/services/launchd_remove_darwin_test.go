//go:build darwin

package services

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/podman"
)

type stopRecorder struct{ stopped []string }

func (s *stopRecorder) Start(string) error                { return nil }
func (s *stopRecorder) Stop(n string) error               { s.stopped = append(s.stopped, n); return nil }
func (s *stopRecorder) Restart(string) error              { return nil }
func (s *stopRecorder) UnitStatus(string) (string, error) { return "inactive", nil }
func (s *stopRecorder) AllUnitStates() map[string]string  { return nil }

// Removing the plist without taking the job out of the domain first left launchd
// holding a job whose file was gone, which is how a removed service went on
// answering `launchctl list` with no plist behind it.
func TestRemoveContainerUnit_stopsTheUnitBeforeDroppingThePlist(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	agents := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(agents, 0o755); err != nil {
		t.Fatal(err)
	}
	plist := filepath.Join(agents, "lerd-kafka.plist")
	if err := os.WriteFile(plist, []byte("<plist/>"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := &stopRecorder{}
	prev := podman.UnitLifecycle
	podman.UnitLifecycle = rec
	t.Cleanup(func() { podman.UnitLifecycle = prev })

	if err := (&darwinServiceManager{}).RemoveContainerUnit("lerd-kafka"); err != nil {
		t.Fatalf("RemoveContainerUnit: %v", err)
	}

	if len(rec.stopped) != 1 || rec.stopped[0] != "lerd-kafka" {
		t.Errorf("stopped = %v, want [lerd-kafka]", rec.stopped)
	}
	if _, err := os.Stat(plist); !os.IsNotExist(err) {
		t.Errorf("plist should be gone, stat err = %v", err)
	}
}

// The guard is the backstop for any path that reaches a mutating verb without a
// stub: there is no per-test launchd domain, so it must panic rather than land on
// the developer's own.
func TestLaunchctl_refusesAMutatingVerbUnderTest(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("a mutating launchctl under test should panic, it returned")
		}
	}()
	_, _ = launchctl("bootstrap", uidDomain(), "/tmp/whatever.plist")
}

// Reads are harmless and several code paths depend on them, so the guard must
// leave them alone.
func TestLaunchctl_allowsAReadUnderTest(t *testing.T) {
	if _, err := launchctl("list"); err != nil {
		t.Errorf("launchctl list should be allowed under test: %v", err)
	}
}
