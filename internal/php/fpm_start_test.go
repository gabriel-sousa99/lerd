package php

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/podman"
)

// stubStartSeams replaces the StartFPM seams for one test and restores them
// afterwards, so no case reaches podman or the real user bus.
type startSeams struct {
	order    []string
	running  []bool
	reported []podman.MountRepair
	repairs  []podman.MountRepair
}

func stubStartSeams(t *testing.T, s *startSeams) {
	t.Helper()
	oldMachine, oldRunning := machineResponsive, containerRunning
	oldInstalled, oldRepair, oldStart := fpmInstalled, repairStaleMounts, startUnit
	oldReporter := MountRepairReporter
	t.Cleanup(func() {
		machineResponsive, containerRunning = oldMachine, oldRunning
		fpmInstalled, repairStaleMounts, startUnit = oldInstalled, oldRepair, oldStart
		MountRepairReporter = oldReporter
	})

	machineResponsive = func() error { return nil }
	containerRunning = func(string) (bool, error) {
		s.order = append(s.order, "running?")
		if len(s.running) == 0 {
			return true, nil
		}
		next := s.running[0]
		s.running = s.running[1:]
		return next, nil
	}
	fpmInstalled = func(string, string) bool { return true }
	repairStaleMounts = func() []podman.MountRepair {
		s.order = append(s.order, "repair")
		return s.repairs
	}
	startUnit = func(string) error {
		s.order = append(s.order, "start")
		return nil
	}
	MountRepairReporter = func(r []podman.MountRepair) { s.reported = append(s.reported, r...) }
}

// A stale bind mount aborts the container start with "statfs <path>: no such
// file or directory" (exit 125) and leaves the unit crash-looping, so the sweep
// has to run before the unit is started, not after it fails (#1083).
func TestStartFPMRepairsStaleMountsBeforeStart(t *testing.T) {
	s := &startSeams{
		running: []bool{false, true},
		repairs: []podman.MountRepair{{
			Unit: "lerd-php84-fpm",
			Path: "/home/dev/.worktrees/app/gone-branch",
			Site: "app",
		}},
	}
	stubStartSeams(t, s)

	if err := StartFPM("8.4", "lerd-php84-fpm"); err != nil {
		t.Fatalf("StartFPM() = %v, want nil", err)
	}

	want := []string{"running?", "repair", "start", "running?"}
	if len(s.order) != len(want) {
		t.Fatalf("call order = %v, want %v", s.order, want)
	}
	for i := range want {
		if s.order[i] != want[i] {
			t.Fatalf("call order = %v, want %v", s.order, want)
		}
	}
	if len(s.reported) != 1 || s.reported[0].Path != "/home/dev/.worktrees/app/gone-branch" {
		t.Errorf("reported = %v, want the dropped worktree mount", s.reported)
	}
}

// Nothing stale means nothing to say: the reporter stays silent so a normal
// start doesn't grow a WARN line.
func TestStartFPMSilentWhenNothingStale(t *testing.T) {
	s := &startSeams{running: []bool{false, true}}
	stubStartSeams(t, s)

	if err := StartFPM("8.4", "lerd-php84-fpm"); err != nil {
		t.Fatalf("StartFPM() = %v, want nil", err)
	}
	if len(s.reported) != 0 {
		t.Errorf("reported = %v, want none", s.reported)
	}
}

// The hot path (container already up) must not touch the quadlets at all:
// RepairMissingMounts writes files and daemon-reloads, which is far too much
// for every `php -v` on a running environment.
func TestStartFPMSkipsRepairWhenAlreadyRunning(t *testing.T) {
	s := &startSeams{running: []bool{true}}
	stubStartSeams(t, s)

	if err := StartFPM("8.4", "lerd-php84-fpm"); err != nil {
		t.Fatalf("StartFPM() = %v, want nil", err)
	}
	for _, step := range s.order {
		if step == "repair" || step == "start" {
			t.Fatalf("call order = %v, want no repair or start", s.order)
		}
	}
}
