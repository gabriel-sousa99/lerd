package ui

import (
	"runtime"
	"testing"
)

// isolateLaunchAgents redirects HOME so a test that writes or removes a unit
// cannot touch the real ~/Library/LaunchAgents. Only macOS derives that
// directory from HOME; the Linux unit dir already follows XDG_CONFIG_HOME.
func isolateLaunchAgents(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "darwin" {
		t.Setenv("HOME", t.TempDir())
	}
}
