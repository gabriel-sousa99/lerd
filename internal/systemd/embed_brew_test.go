package systemd

import (
	"strings"
	"testing"
)

// Homebrew installs the binary into a version-pinned Cellar directory and links
// the current version into opt/. A unit that recorded the Cellar path stops
// working on the next `brew upgrade lerd`, which is why config.LerdBinary hands
// back the opt spelling for units to carry.
func TestGetUnitKeepsStableBrewPath(t *testing.T) {
	for _, brewPath := range []string{
		"/opt/homebrew/opt/lerd/bin/lerd",
		"/home/linuxbrew/.linuxbrew/opt/lerd/bin/lerd",
	} {
		prev := lerdBinaryPath
		lerdBinaryPath = func() string { return brewPath }
		unit, err := GetUnit("lerd-ui")
		lerdBinaryPath = prev
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(unit, "ExecStart="+brewPath+" serve-ui") {
			t.Errorf("unit does not run %s:\n%s", brewPath, unit)
		}
	}
}
