package ui

import (
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/serviceops"
)

// The fix log is where a user on a metered connection finds out a doctor fix
// is about to download something, so the announced pull carries its size.
func TestInstallPhaseLineDisclosesTheDownloadSize(t *testing.T) {
	line := installPhaseLine("redis", serviceops.PhaseEvent{
		Phase: "pulling_image",
		Image: "docker.io/library/redis:7-alpine",
		Bytes: 18 * 1024 * 1024,
	})
	if !strings.Contains(line, "docker.io/library/redis:7-alpine") || !strings.Contains(line, "18.0 MiB") {
		t.Errorf("unexpected line: %q", line)
	}
}

func TestInstallPhaseLineWithoutASizeReadsNormally(t *testing.T) {
	line := installPhaseLine("redis", serviceops.PhaseEvent{
		Phase: "pulling_image",
		Image: "docker.io/library/redis:7-alpine",
	})
	if line != "redis: pulling docker.io/library/redis:7-alpine" {
		t.Errorf("unexpected line: %q", line)
	}
}
