package php

import (
	"errors"
	"fmt"
	"time"

	"github.com/gabriel-sousa99/lerd/internal/podman"
	"github.com/gabriel-sousa99/lerd/internal/services"
)

// ErrFPMNotInstalled is returned (wrapped) by StartFPM when the requested
// version's FPM container does not exist yet, so the caller knows the fix is to
// install it rather than just start it. Test with errors.Is.
var ErrFPMNotInstalled = errors.New("FPM container is not installed")

// Seams for the start path, swappable in tests.
var (
	machineResponsive = podman.EnsureMachineResponsive
	containerRunning  = podman.ContainerRunning
	fpmInstalled      = FPMInstalled
	repairStaleMounts = podman.RepairMissingMounts
	startUnit         = podman.StartUnit
)

// MountRepairReporter, when set, receives the stale bind mounts the start-time
// preflight dropped, so a CLI caller can warn about them. StartFPM itself stays
// silent (its callers own all user-facing output), and the MCP path leaves this
// nil.
var MountRepairReporter func([]podman.MountRepair)

// FPMInstalled reports whether the FPM container already exists (so it only
// needs starting) rather than needing a fresh build. It checks both the shared
// per-version units and the specific container's unit, so a custom-FPM container
// (lerd-cfpm-<site>, whose version isn't in the shared list) is recognised too.
func FPMInstalled(version, container string) bool {
	return IsInstalled(version) || services.Mgr.ContainerUnitInstalled(container)
}

// StartFPM brings up a stopped-but-installed FPM container and waits for it to
// report running, so an exec right after doesn't race the boot. It is a no-op
// when the container is already running and returns ErrFPMNotInstalled (wrapped
// with the version) when the container doesn't exist yet. It produces no output;
// callers add any user-facing progress. Shared by the CLI php/artisan/shell
// commands and the MCP exec handlers so both auto-start the same way.
func StartFPM(version, container string) error {
	// Gate on a bounded machine probe so a post-sleep stall surfaces (and self-
	// heals) fast instead of hanging the caller on the untimed inspect below.
	// Once this passes the VM is responsive, so the exec that follows won't hang.
	if err := machineResponsive(); err != nil {
		return err
	}
	if running, _ := containerRunning(container); running {
		return nil
	}
	if !fpmInstalled(version, container) {
		return fmt.Errorf("PHP %s: %w", version, ErrFPMNotInstalled)
	}
	// Pre-flight: a bind mount whose host directory is gone (a removed
	// worktree, a deleted project) makes podman refuse the start with
	// "statfs <path>: no such file or directory" and exit 125, which leaves
	// the unit crash-looping and every site on that version down (#1083).
	// `lerd start` already sweeps the quadlets; on-demand starts — the php /
	// artisan / console shims and the MCP exec handlers, which are what most
	// people actually touch first — used to walk straight into the failure
	// with no way out but editing the quadlet by hand.
	if repairs := repairStaleMounts(); len(repairs) > 0 && MountRepairReporter != nil {
		MountRepairReporter(repairs)
	}
	if err := startUnit(container); err != nil {
		return fmt.Errorf("starting PHP %s FPM: %w", version, err)
	}
	deadline := time.Now().Add(20 * time.Second)
	for {
		if running, _ := containerRunning(container); running {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s to start", container)
		}
		time.Sleep(500 * time.Millisecond)
	}
}
