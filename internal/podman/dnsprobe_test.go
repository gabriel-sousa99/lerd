package podman

import (
	"os/exec"
	"testing"
	"time"
)

// getent exits 0 only when the name actually resolved, so its status is the
// whole answer: aardvark-dns forwarded the query and got a record back.
func TestResolvesFromNginx_TrueOnlyWhenGetentSucceeds(t *testing.T) {
	prev := execCommand
	t.Cleanup(func() { execCommand = prev })

	execCommand = fakeExec("140.82.121.4  example.test\n", "", 0)
	if !ResolvesFromNginx("example.test") {
		t.Error("a getent that exits 0 means the container resolved the name")
	}

	execCommand = fakeExec("", "", 2)
	if ResolvesFromNginx("example.test") {
		t.Error("a getent that exits non-zero means the lookup failed, not that it passed")
	}
}

// An empty name would make getent list the whole hosts database and exit 0,
// which would report a healthy resolver on a store base that carries no host.
func TestResolvesFromNginx_EmptyNameNeverPasses(t *testing.T) {
	prev := execCommand
	t.Cleanup(func() { execCommand = prev })
	execCommand = func(string, ...string) *exec.Cmd {
		t.Error("an empty name must not reach podman exec")
		return exec.Command("true")
	}

	if ResolvesFromNginx("") {
		t.Error("ResolvesFromNginx(\"\") = true, want false")
	}
}

// A broken forwarder is exactly the case where the lookup hangs instead of
// failing, so doctor must come back on its own rather than sit there.
func TestResolvesFromNginx_CapsWallClock(t *testing.T) {
	prevExec, prevTimeout := execCommand, dnsProbeTimeout
	t.Cleanup(func() { execCommand, dnsProbeTimeout = prevExec, prevTimeout })
	dnsProbeTimeout = 50 * time.Millisecond
	execCommand = func(string, ...string) *exec.Cmd {
		return exec.Command("sleep", "30")
	}

	done := make(chan bool, 1)
	go func() { done <- ResolvesFromNginx("example.test") }()

	select {
	case got := <-done:
		if got {
			t.Error("a probe killed on timeout must report failure, not success")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no wall-clock cap; a stalled resolver hangs lerd doctor forever")
	}
}
