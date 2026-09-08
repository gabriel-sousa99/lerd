package workerheal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/siteinfo"
)

// stubFinishedWorker points detection at one site whose only unit is inactive
// and enabled, with the Restart policy under test.
func stubFinishedWorker(t *testing.T, restart string) []UnhealthyWorker {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	lerdDir := filepath.Join(dir, "lerd")
	if err := os.MkdirAll(lerdDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sites := "sites:\n  - name: demo\n    path: " + dir + "\n    domains: [demo.test]\n"
	if err := os.WriteFile(filepath.Join(lerdDir, "sites.yaml"), []byte(sites), 0o644); err != nil {
		t.Fatal(err)
	}

	prevStopped, prevStates, prevMeta, prevEnabled := isStoppedFn, unitStatesFn, unitMetaFn, unitEnabledFn
	t.Cleanup(func() {
		isStoppedFn, unitStatesFn, unitMetaFn, unitEnabledFn = prevStopped, prevStates, prevMeta, prevEnabled
	})
	isStoppedFn = func() bool { return false }
	unitStatesFn = func() map[string]string {
		return map[string]string{"lerd-native-demo.service": "inactive"}
	}
	unitMetaFn = func() map[string]siteinfo.UnitMeta {
		return map[string]siteinfo.UnitMeta{"lerd-native-demo.service": {Restart: restart}}
	}
	unitEnabledFn = func(string) bool { return true }

	out, err := Detect()
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// Closing the window a worker was running ends its command cleanly. Its
// definition says so by declaring on-failure, and reporting that as drift asks
// the user to heal something they closed on purpose.
func TestDetect_onFailureWorkerThatFinishedIsNotDrift(t *testing.T) {
	if got := stubFinishedWorker(t, "on-failure"); len(got) != 0 {
		t.Errorf("Detect = %+v, want nothing to heal", got)
	}
}

// An always-restart unit sitting inactive is inactive against its own policy,
// which is the drift this check exists for.
func TestDetect_alwaysWorkerStoppedIsStillDrift(t *testing.T) {
	got := stubFinishedWorker(t, "always")
	if len(got) != 1 || got[0].State != "expected-but-stopped" {
		t.Errorf("Detect = %+v, want one expected-but-stopped worker", got)
	}
}

// A systemd too old to report the property leaves it empty, and the check has to
// keep behaving the way it did rather than go quiet everywhere.
func TestDetect_unknownRestartKeepsTheOldBehaviour(t *testing.T) {
	if got := stubFinishedWorker(t, ""); len(got) != 1 {
		t.Errorf("Detect = %+v, want the drift report to survive a missing property", got)
	}
}
