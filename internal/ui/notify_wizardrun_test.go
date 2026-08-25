package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gabriel-sousa99/lerd/internal/push"
)

// stubRunNotifier records what a finished run would raise.
func stubRunNotifier(t *testing.T) *[]push.Notification {
	t.Helper()
	var sent []push.Notification
	original := dispatchRunNotification
	dispatchRunNotification = func(n push.Notification) { sent = append(sent, n) }
	t.Cleanup(func() { dispatchRunNotification = original })
	return &sent
}

// Scaffolding is the run the wizard exists to send to the background, so its
// ending has to reach the user with no dashboard page open.
func TestNotificationForScaffoldRun(t *testing.T) {
	n, ok := notificationForRun(runSnapshot{
		Kind:   runKindScaffold,
		Dir:    "/home/u/projects",
		Label:  "/home/u/projects/shop",
		Status: runDone,
	}, time.Now())
	if !ok {
		t.Fatal("a finished scaffold should notify")
	}
	if n.Kind != "op_done" {
		t.Errorf("kind = %q, want op_done", n.Kind)
	}
	if !strings.Contains(n.Title, "shop") {
		t.Errorf("title = %q, want it to name the project", n.Title)
	}
}

// A step that broke is worth interrupting for whichever kind of run it was.
func TestNotificationForFailedRun(t *testing.T) {
	n, ok := notificationForRun(runSnapshot{
		Kind:   runKindSetup,
		Dir:    "/home/u/projects/shop",
		Status: runFailed,
		Error:  "composer could not resolve dependencies",
	}, time.Now())
	if !ok {
		t.Fatal("a failed run should notify")
	}
	if n.Kind != "op_failed" {
		t.Errorf("kind = %q, want op_failed", n.Kind)
	}
	if !strings.Contains(n.Body, "resolve dependencies") {
		t.Errorf("body = %q, want the reason it failed", n.Body)
	}
	if !strings.Contains(n.Title, "shop") {
		t.Errorf("title = %q, want it to name the project", n.Title)
	}
}

// A full setup is five or six quick steps. One notification each would bury the
// ones that matter, and the wizard is already showing them on screen.
func TestQuickRunsStayQuiet(t *testing.T) {
	for _, kind := range []string{runKindLink, runKindEnv, runKindSetup} {
		if _, ok := notificationForRun(runSnapshot{Kind: kind, Dir: "/home/u/shop", Status: runDone}, time.Now()); ok {
			t.Errorf("a finished %s run should not notify", kind)
		}
	}
}

// The registry raises it itself, so nothing depends on a page still watching.
func TestRunRegistryNotifiesOnFailure(t *testing.T) {
	sent := stubRunNotifier(t)
	stubRunExec(t, func(r *run) error {
		r.append("could not write to disk")
		return errors.New("exit status 1")
	})

	reg := newRunRegistry()
	r := reg.Start(runKindScaffold, t.TempDir(), "/tmp/shop", []string{"lerd", "new"})
	waitForStatus(t, r, runFailed)

	deadline := time.Now().Add(time.Second)
	for len(*sent) == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if len(*sent) != 1 {
		t.Fatalf("raised %d notifications, want 1", len(*sent))
	}
	if (*sent)[0].Kind != "op_failed" {
		t.Errorf("kind = %q, want op_failed", (*sent)[0].Kind)
	}
}

// A finished scaffold reports itself the same way, so a project created while
// the dashboard was closed is not silently waiting.
func TestRunRegistryNotifiesOnScaffoldSuccess(t *testing.T) {
	sent := stubRunNotifier(t)
	stubRunExec(t, func(_ *run) error { return nil })

	reg := newRunRegistry()
	r := reg.Start(runKindScaffold, t.TempDir(), "/tmp/shop", []string{"lerd", "new"})
	waitForStatus(t, r, runDone)

	deadline := time.Now().Add(time.Second)
	for len(*sent) == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if len(*sent) != 1 || (*sent)[0].Kind != "op_done" {
		t.Fatalf("raised %+v, want one op_done", *sent)
	}
}
