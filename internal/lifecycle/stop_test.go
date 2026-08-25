package lifecycle

import (
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/podman"
)

// recorder captures the teardown as an ordered list of steps, so a test can
// assert both what was stopped and what order it happened in.
type recorder struct {
	mu    sync.Mutex
	steps []string
}

func (r *recorder) add(step string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.steps = append(r.steps, step)
}

func (r *recorder) index(step string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Index(r.steps, step)
}

func (r *recorder) contains(step string) bool { return r.index(step) >= 0 }

// captureTeardown isolates config under a temp HOME and swaps the three side
// effects (unit stops, the batch container stop, the VM stop) for recording
// stubs. Returns the recorder holding the steps in the order they ran.
func captureTeardown(t *testing.T) *recorder {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, ".local", "share"))

	rec := &recorder{}
	origStopUnit, origMachine, origBatch := stopUnitFn, stopPodmanMachineFn, batchStopFn
	t.Cleanup(func() {
		stopUnitFn, stopPodmanMachineFn, batchStopFn = origStopUnit, origMachine, origBatch
	})
	stopUnitFn = func(name string) error { rec.add(name); return nil }
	stopPodmanMachineFn = func() { rec.add("podman-machine") }
	batchStopFn = func([]string) { rec.add("batch-containers") }
	return rec
}

// TestShutdownForLogout_NeverStopsWatcherUnit is the regression test for the
// deadlock this path was written to avoid. Stopping lerd-watcher asks launchd
// to bootout the job the watcher is running inside; bootout blocks until that
// process exits, so the teardown would hang until launchd's exit timeout fired
// and SIGKILLed it, leaving the Podman Machine VM running.
func TestShutdownForLogout_NeverStopsWatcherUnit(t *testing.T) {
	rec := captureTeardown(t)

	if err := ShutdownForLogout(SimpleRunner); err != nil {
		t.Fatalf("ShutdownForLogout: %v", err)
	}
	if rec.contains(podman.WatcherUnit) {
		t.Errorf("shutdown handler attempted to stop %s, which deadlocks against its own launchd bootout", podman.WatcherUnit)
	}
}

// TestShutdownForLogout_StopsVMBeforeHostProcesses pins the ordering that makes
// the feature worth having. The VM stop is the step whose loss corrupts data,
// so it must not sit behind host processes that launchd is terminating anyway.
func TestShutdownForLogout_StopsVMBeforeHostProcesses(t *testing.T) {
	rec := captureTeardown(t)

	if err := ShutdownForLogout(SimpleRunner); err != nil {
		t.Fatalf("ShutdownForLogout: %v", err)
	}

	machine := rec.index("podman-machine")
	if machine < 0 {
		t.Fatal("Podman Machine VM was never stopped, which is the entire point of the logout teardown")
	}
	if batch := rec.index("batch-containers"); batch < 0 || batch > machine {
		t.Error("containers must stop before the VM they run inside")
	}
	for _, unit := range QuitProcessUnits(podman.WatcherUnit) {
		if at := rec.index(unit); at >= 0 && at < machine {
			t.Errorf("%s stopped before the VM; host processes must not delay the VM stop", unit)
		}
	}
}

// TestShutdownForLogout_StopsTheOtherProcessUnits pins that skipping ourselves
// does not skip the rest of the session.
func TestShutdownForLogout_StopsTheOtherProcessUnits(t *testing.T) {
	rec := captureTeardown(t)

	if err := ShutdownForLogout(SimpleRunner); err != nil {
		t.Fatalf("ShutdownForLogout: %v", err)
	}
	for _, unit := range []string{"lerd-ui", "lerd-tray", "lerd-dns"} {
		if !rec.contains(unit) {
			t.Errorf("logout teardown left %s running", unit)
		}
	}
}

// TestShutdownForLogout_MarksStopped pins a deliberate product decision: a
// logout leaves the intentional-stop marker set. Between our signal and actual
// power-off, lerd-ui's health watcher may still be alive and would read every
// unit we are stopping as a crash, firing heals and notifications that race the
// teardown. Autostart clears the marker again on the next boot, since a fresh
// lerd-ui calls config.ClearStopped (internal/ui/worker_health_watcher.go).
func TestShutdownForLogout_MarksStopped(t *testing.T) {
	captureTeardown(t)

	if err := ShutdownForLogout(SimpleRunner); err != nil {
		t.Fatalf("ShutdownForLogout: %v", err)
	}
	if !config.IsStopped() {
		t.Error("logout teardown must mark lerd stopped so heal watchers stay quiet during shutdown")
	}
}

// TestQuit_StopsWatcherAndVMLast pins that the CLI path is unchanged by the
// watcher's special casing: `lerd quit` still tears down every process unit,
// the watcher included, and stops the VM at the very end.
func TestQuit_StopsWatcherAndVMLast(t *testing.T) {
	rec := captureTeardown(t)

	if err := Quit(SimpleRunner, nil); err != nil {
		t.Fatalf("Quit: %v", err)
	}
	if !rec.contains(podman.WatcherUnit) {
		t.Errorf("`lerd quit` must still stop %s", podman.WatcherUnit)
	}
	machine := rec.index("podman-machine")
	if machine < 0 {
		t.Fatal("`lerd quit` must stop the Podman Machine VM")
	}
	for _, unit := range QuitProcessUnits() {
		if at := rec.index(unit); at > machine {
			t.Errorf("%s stopped after the VM; `lerd quit` stops processes first", unit)
		}
	}
}

// TestStop_LeavesDNSAndProcessUnitsAlone pins that `lerd stop` is not a
// teardown of the session: dns stays up as install-level plumbing and the VM
// keeps running.
func TestStop_LeavesDNSAndProcessUnitsAlone(t *testing.T) {
	rec := captureTeardown(t)

	if err := Stop(SimpleRunner); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if rec.contains("lerd-dns") {
		t.Error("`lerd stop` must leave lerd-dns up so the resolver keeps resolving .test")
	}
	if rec.contains("podman-machine") {
		t.Error("`lerd stop` must not stop the Podman Machine VM")
	}
}

// TestQuit_RunsTheHostCleanupBeforeTheVMStop pins that `lerd quit` clears a
// directly-launched tray before `podman machine stop`, not after. The VM stop
// takes seconds, and a tray icon still on screen through it reads as a hung
// quit.
func TestQuit_RunsTheHostCleanupBeforeTheVMStop(t *testing.T) {
	rec := captureTeardown(t)

	if err := Quit(SimpleRunner, func() { rec.add("host-cleanup") }); err != nil {
		t.Fatalf("Quit: %v", err)
	}

	cleanup, machine := rec.index("host-cleanup"), rec.index("podman-machine")
	if cleanup < 0 {
		t.Fatal("Quit never ran the host cleanup hook")
	}
	if cleanup > machine {
		t.Error("host cleanup ran after the VM stop, so it waited out the whole machine stop")
	}
}

// TestQuit_NilHookIsSkipped pins that the hook is optional, so the logout path
// and the tests can pass nil without Quit panicking.
func TestQuit_NilHookIsSkipped(t *testing.T) {
	rec := captureTeardown(t)

	if err := Quit(SimpleRunner, nil); err != nil {
		t.Fatalf("Quit: %v", err)
	}
	if !rec.contains("podman-machine") {
		t.Error("Quit must still stop the VM with no hook set")
	}
}
