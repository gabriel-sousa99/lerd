// Package imagepull discloses the container images a command is about to
// download, estimates their size from the registry without pulling, and
// enforces the offline mode that lets a user on a metered connection skip
// every pull and rebuild that is not strictly required.
package imagepull

import (
	"os"
	"strings"
	"sync/atomic"
)

var (
	offlineFlag atomic.Bool
	dryRunFlag  atomic.Bool
)

// SetOffline records the --no-pull flag. LERD_OFFLINE sets the same mode for
// callers that never parse flags (the daemon, the watcher, the MCP server).
func SetOffline(v bool) { offlineFlag.Store(v) }

// Offline reports whether lerd should skip pulls and rebuilds unless the
// image is missing outright.
func Offline() bool {
	return offlineFlag.Load() || truthy(os.Getenv("LERD_OFFLINE"))
}

// SetDryRun records the --dry-run flag.
func SetDryRun(v bool) { dryRunFlag.Store(v) }

// DryRun reports whether the command should report what it would download
// and stop before downloading it.
func DryRun() bool { return dryRunFlag.Load() }

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}
