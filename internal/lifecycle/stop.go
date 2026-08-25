package lifecycle

import (
	"io"
	"strings"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/feedback"
	"github.com/gabriel-sousa99/lerd/internal/podman"
)

// Swappable for tests so the teardown order can be asserted without stopping
// real units or shelling out to podman.
var (
	stopUnitFn          = podman.StopUnit
	stopPodmanMachineFn = StopPodmanMachine
	batchStopFn         = BatchStopContainers
)

// Stop tears down every container, service, and worker unit, running the
// per-unit stops through runner. skip removes units the caller must not stop.
func Stop(runner ParallelRunner, skip ...string) error {
	units := StopUnitSet(skip...)

	feedback.Begin()
	feedback.Line("stopping lerd")

	// Mark the intentional shutdown before tearing anything down, so the worker
	// health watcher (which keeps running) suppresses heal/notification noise for
	// the workers we're about to stop. They stay enabled and come back on start.
	_ = config.MarkStopped()

	// On macOS: stop all containers in one podman call before the parallel
	// per-unit jobs run. This avoids serialising N individual podman stop
	// requests through the Podman Machine socket (which can take 5s × N).
	batchStopFn(units)

	jobs := make([]Job, len(units))
	for i, u := range units {
		unit := u
		label := strings.TrimSuffix(strings.TrimPrefix(unit, "lerd-"), ".timer")
		jobs[i] = Job{
			Label: label,
			Run:   func(w io.Writer) error { return stopUnitFn(unit) },
		}
	}
	_ = runner(jobs)
	return nil
}

// Quit is the full teardown behind `lerd quit`: everything Stop covers, then
// the host process units, then the Podman Machine VM. skip removes units the
// caller must not stop. beforeMachineStop, when set, runs after the process
// units and before the VM, for host cleanup that must not wait out a VM stop
// that takes seconds; pass nil when there is none.
func Quit(runner ParallelRunner, beforeMachineStop func(), skip ...string) error {
	if err := Stop(runner, skip...); err != nil {
		return err
	}
	stopProcessUnits(QuitProcessUnits(skip...))
	if beforeMachineStop != nil {
		beforeMachineStop()
	}
	stopPodmanMachineFn()
	return nil
}

// ShutdownForLogout is the teardown the watcher runs when the OS is logging out
// or restarting. It differs from Quit in two ways that both matter only when
// the shutdown is driven from inside lerd-watcher.
//
// It never stops podman.WatcherUnit, and it stops the Podman Machine VM before the
// host process units rather than after. The VM is the only step whose loss
// costs anything: a database killed mid-write replays its write-ahead log for
// minutes on the next start. lerd-ui, lerd-dns and lerd-tray are host processes
// that launchd is terminating anyway, so they are the right thing to leave
// until last if the exit grace runs out.
func ShutdownForLogout(runner ParallelRunner) error {
	if err := Stop(runner, podman.WatcherUnit); err != nil {
		return err
	}
	stopPodmanMachineFn()
	stopProcessUnits(QuitProcessUnits(podman.WatcherUnit))
	return nil
}

// stopProcessUnits stops host process units in order, reporting each one.
func stopProcessUnits(units []string) {
	for _, unit := range units {
		s := feedback.Start("stopping " + unit)
		if err := stopUnitFn(unit); err != nil {
			s.Fail(err)
		} else {
			s.OK("")
		}
	}
}
