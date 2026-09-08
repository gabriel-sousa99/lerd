package podman

import (
	"errors"
	"io"
	"os/exec"
	"strings"
	"testing"
)

// recordExec returns an execCommand stub that appends every invocation's args
// and answers with the given per-call exit codes.
func recordExec(calls *[][]string, exits ...int) func(string, ...string) *exec.Cmd {
	n := 0
	return func(name string, args ...string) *exec.Cmd {
		*calls = append(*calls, args)
		exit := 0
		if n < len(exits) {
			exit = exits[n]
		}
		n++
		return fakeExec("", "", exit)(name, args...)
	}
}

func hasDNSFlag(args []string, server string) bool {
	for i, a := range args {
		if a == "--dns" && i+1 < len(args) && args[i+1] == server {
			return true
		}
	}
	return false
}

func hasHostNetwork(args []string) bool {
	for i, a := range args {
		if a == "--network" && i+1 < len(args) && args[i+1] == "host" {
			return true
		}
	}
	return false
}

// The plain build is what works on a healthy host, so a passing one must not
// drag the host's resolvers into the image build.
func TestBuildDNSMasqImage_NoRetryWhenTheFirstBuildPasses(t *testing.T) {
	prev := execCommand
	t.Cleanup(func() { execCommand = prev })
	var calls [][]string
	execCommand = recordExec(&calls, 0)

	if err := BuildDNSMasqImage(io.Discard, []string{"192.0.2.1"}); err != nil {
		t.Fatalf("BuildDNSMasqImage() = %v, want nil", err)
	}
	if len(calls) != 1 {
		t.Fatalf("ran %d builds, want 1", len(calls))
	}
	if hasDNSFlag(calls[0], "192.0.2.1") {
		t.Error("the first build must run with podman's own resolver handling, not pinned servers")
	}
}

// apk resolves from inside the build's network namespace, where the host's
// stub resolver is not reachable. Pinning the real upstreams is the retry.
func TestBuildDNSMasqImage_RetriesWithTheHostResolvers(t *testing.T) {
	prev := execCommand
	t.Cleanup(func() { execCommand = prev })
	var calls [][]string
	execCommand = recordExec(&calls, 1, 0)

	if err := BuildDNSMasqImage(io.Discard, []string{"192.0.2.1", "192.0.2.2"}); err != nil {
		t.Fatalf("BuildDNSMasqImage() = %v, want nil once the retry succeeds", err)
	}
	if len(calls) != 2 {
		t.Fatalf("ran %d builds, want 2", len(calls))
	}
	for _, server := range []string{"192.0.2.1", "192.0.2.2"} {
		if !hasDNSFlag(calls[1], server) {
			t.Errorf("retry args %v carry no --dns %s", calls[1], server)
		}
	}
	if calls[1][len(calls[1])-1] != "-" {
		t.Errorf("retry args %v must still read the Containerfile from stdin", calls[1])
	}
}

// With nothing to pin, the pinned retry has no servers to use, so the host
// network is the only fallback left.
func TestBuildDNSMasqImage_FallsBackToTheHostNetworkWithoutNameservers(t *testing.T) {
	prev := execCommand
	t.Cleanup(func() { execCommand = prev })
	var calls [][]string
	execCommand = recordExec(&calls, 1, 0)

	if err := BuildDNSMasqImage(io.Discard, nil); err != nil {
		t.Fatalf("BuildDNSMasqImage() = %v, want nil once the host-network build succeeds", err)
	}
	if len(calls) != 2 {
		t.Fatalf("ran %d builds, want 2", len(calls))
	}
	if !hasHostNetwork(calls[1]) {
		t.Errorf("fallback args %v do not build on the host network", calls[1])
	}
}

// The reported host resolves nothing from inside the build namespace, pinned
// servers included, so the host network has to be tried after the pinned retry.
func TestBuildDNSMasqImage_FallsBackToTheHostNetworkAfterThePinnedRetry(t *testing.T) {
	prev := execCommand
	t.Cleanup(func() { execCommand = prev })
	var calls [][]string
	execCommand = recordExec(&calls, 1, 1, 0)

	if err := BuildDNSMasqImage(io.Discard, []string{"192.0.2.1"}); err != nil {
		t.Fatalf("BuildDNSMasqImage() = %v, want nil once the host-network build succeeds", err)
	}
	if len(calls) != 3 {
		t.Fatalf("ran %d builds, want 3", len(calls))
	}
	if hasDNSFlag(calls[2], "192.0.2.1") {
		t.Error("the host-network build must not also pin resolvers the namespace could not reach")
	}
	if !hasHostNetwork(calls[2]) {
		t.Errorf("fallback args %v do not build on the host network", calls[2])
	}
	if calls[2][len(calls[2])-1] != "-" {
		t.Errorf("fallback args %v must still read the Containerfile from stdin", calls[2])
	}
}

// When even the host network cannot build, the caller has to see the real
// failure rather than a wrapped one.
func TestBuildDNSMasqImage_ReturnsTheLastFailure(t *testing.T) {
	prev := execCommand
	t.Cleanup(func() { execCommand = prev })
	var calls [][]string
	execCommand = recordExec(&calls, 1, 1, 1)

	err := BuildDNSMasqImage(io.Discard, []string{"192.0.2.1"})
	if err == nil {
		t.Fatal("BuildDNSMasqImage() = nil, want the build error")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Errorf("err = %v, want the underlying build failure", err)
	}
	if len(calls) != 3 {
		t.Fatalf("ran %d builds, want 3", len(calls))
	}
}

// Every build path must produce the same image, or `lerd start` rebuilds what
// `lerd install` just built.
func TestBuildDNSMasqImage_TagsTheImageLerdDnsmasq(t *testing.T) {
	prev := execCommand
	t.Cleanup(func() { execCommand = prev })
	var calls [][]string
	execCommand = recordExec(&calls, 0)

	_ = BuildDNSMasqImage(io.Discard, nil)
	if !strings.Contains(strings.Join(calls[0], " "), "-t "+DNSMasqImage) {
		t.Errorf("build args %v do not tag %s", calls[0], DNSMasqImage)
	}
}
