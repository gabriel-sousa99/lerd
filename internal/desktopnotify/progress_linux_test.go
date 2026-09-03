//go:build linux

package desktopnotify

import "testing"

// A session with no notification daemon still runs the work; the progress popup
// is what degrades, not the start behind it.
func TestStartProgress_withoutABusIsANoop(t *testing.T) {
	withoutSessionBus(t)

	p := StartProgress("Starting Lerd", "Preparing")
	p.Step("lerd-mysql")
	p.Percent(1, 4)
	p.Close()
	if _, ok := p.(noProgress); !ok {
		t.Errorf("StartProgress = %T, want the no-op", p)
	}
}

// The bar is a percentage of the units the run announced, capped so a run that
// starts more than it counted cannot overshoot.
func TestBusProgress_percentIsCappedAndIgnoresAnUnknownTotal(t *testing.T) {
	// Percent pushes, and a bus left connected here posts a real popup on the
	// desktop of whoever runs the suite, replacing an id that is not theirs.
	withoutSessionBus(t)

	p := &busProgress{id: 1}
	p.Percent(1, 4)
	if p.percent != 25 {
		t.Errorf("percent = %d, want 25", p.percent)
	}
	p.Percent(9, 4)
	if p.percent != 100 {
		t.Errorf("percent = %d, want it capped at 100", p.percent)
	}
	p.Percent(3, 0)
	if p.percent != 100 {
		t.Errorf("percent = %d, want an unknown total to leave it alone", p.percent)
	}
}

// Close is called from a defer and may follow a failed start, so a second call
// must not reach the bus again.
func TestBusProgress_closeIsIdempotent(t *testing.T) {
	p := &busProgress{id: 0}
	p.Close()
	p.Close()
	if !p.closed {
		t.Error("Close did not mark the popup closed")
	}
	p.Step("ignored")
	if p.body != "" {
		t.Errorf("body = %q, want a closed popup to ignore updates", p.body)
	}
}

// withoutSessionBus points the package at a bus that cannot be dialled, so a
// test that reaches push() degrades instead of talking to the session running
// the suite. The cached connection is cleared and put back afterwards.
func withoutSessionBus(t *testing.T) {
	t.Helper()
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "unix:path=/nonexistent/lerd-test-bus")
	busMu.Lock()
	prev := busConn
	busConn = nil
	busMu.Unlock()
	t.Cleanup(func() {
		busMu.Lock()
		busConn = prev
		busMu.Unlock()
	})
}
