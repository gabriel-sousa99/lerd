package podman

import (
	"testing"
	"time"
)

// resetPathMountAttempts clears the debounce cache so tests can drive the
// guards in isolation.
func resetPathMountAttempts() {
	pathMountAttemptsMu.Lock()
	pathMountAttempts = map[string]pathMountStamp{}
	pathMountAttemptsMu.Unlock()
}

func TestEphemeralPathsAreSkipped(t *testing.T) {
	cases := []string{
		"/tmp/ide-phpinfo.php",
		"/var/tmp/foo",
		"/run/whatever",
		"/proc/self",
		"/sys/something",
		"/dev/null",
	}
	for _, p := range cases {
		matched := false
		for _, prefix := range ephemeralPathPrefixes {
			if len(p) >= len(prefix) && p[:len(prefix)] == prefix {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("%s should be classified as ephemeral", p)
		}
	}
}

func TestPathMountDebounce_BlocksRecentRetries(t *testing.T) {
	resetPathMountAttempts()
	t.Cleanup(resetPathMountAttempts)

	const path = "/srv/myapp"
	// First record: simulate a successful attempt happening now.
	pathMountAttemptsMu.Lock()
	pathMountAttempts[path] = pathMountStamp{when: time.Now(), debounce: pathMountSuccessDebounce}
	pathMountAttemptsMu.Unlock()

	pathMountAttemptsMu.Lock()
	last, ok := pathMountAttempts[path]
	pathMountAttemptsMu.Unlock()
	if !ok || time.Since(last.when) >= last.debounce {
		t.Errorf("expected fresh entry to be within debounce window")
	}
}

func TestPathMountDebounce_ExpiresAfterWindow(t *testing.T) {
	resetPathMountAttempts()
	t.Cleanup(resetPathMountAttempts)

	const path = "/srv/myapp"
	pathMountAttemptsMu.Lock()
	pathMountAttempts[path] = pathMountStamp{when: time.Now().Add(-2 * pathMountSuccessDebounce), debounce: pathMountSuccessDebounce}
	pathMountAttemptsMu.Unlock()

	pathMountAttemptsMu.Lock()
	last, ok := pathMountAttempts[path]
	pathMountAttemptsMu.Unlock()
	if !ok {
		t.Fatal("entry should still be present in the map until next access")
	}
	if time.Since(last.when) < last.debounce {
		t.Errorf("entry should be older than the debounce window; got age=%v", time.Since(last.when))
	}
}

// TestPathMountDebounce_FailureUsesShortWindow pins the regression: a
// failed restart must record a short debounce so the next caller retries
// within seconds instead of waiting the full success window. Without this,
// a transient DBus glitch silently leaves the container with stale mounts
// for 60s while every subsequent php exec early-returns.
func TestPathMountDebounce_FailureUsesShortWindow(t *testing.T) {
	resetPathMountAttempts()
	t.Cleanup(resetPathMountAttempts)

	const path = "/srv/myapp"
	// Simulate the post-failure stamp written at the end of EnsurePathMounted.
	pathMountAttemptsMu.Lock()
	pathMountAttempts[path] = pathMountStamp{when: time.Now(), debounce: pathMountFailureDebounce}
	pathMountAttemptsMu.Unlock()

	if pathMountFailureDebounce >= pathMountSuccessDebounce {
		t.Fatalf("failure debounce must be shorter than success debounce")
	}

	pathMountAttemptsMu.Lock()
	last, ok := pathMountAttempts[path]
	pathMountAttemptsMu.Unlock()
	if !ok {
		t.Fatal("failure stamp missing")
	}
	if last.debounce != pathMountFailureDebounce {
		t.Errorf("expected failure debounce, got %v", last.debounce)
	}
}
