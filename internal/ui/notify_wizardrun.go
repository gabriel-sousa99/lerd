package ui

import (
	"errors"
	"path/filepath"
	"time"

	"github.com/gabriel-sousa99/lerd/internal/push"
)

// runOpTitles names each kind of run the way a notification should read.
var runOpTitles = map[string]string{
	runKindScaffold: "Create",
	runKindLink:     "Link",
	runKindEnv:      "Environment setup",
	runKindSetup:    "Setup",
}

// notifyRunFinished reports a background run the way every other long operation
// is reported: through the daemon, so it reaches the desktop or a subscribed
// device whether or not a dashboard page is open.
//
// Not every run is worth a notification. A scaffold is minutes of composer and
// the reason the wizard can be sent to the background at all, and a failure is
// always worth interrupting for; the quick steps in between would fire one
// notification each and drown the useful ones.
func notificationForRun(snap runSnapshot, start time.Time) (push.Notification, bool) {
	failed := snap.Status == runFailed
	if !failed && snap.Kind != runKindScaffold {
		return push.Notification{}, false
	}

	opTitle, known := runOpTitles[snap.Kind]
	if !known {
		return push.Notification{}, false
	}

	// A scaffold's label is the directory it creates; everything else is about
	// the project it runs in.
	path := snap.Dir
	if snap.Kind == runKindScaffold && snap.Label != "" {
		path = snap.Label
	}
	project := filepath.Base(path)

	var runErr error
	if failed {
		runErr = errors.New(snap.Error)
	}
	return opNotification(opTitle, project, "lerd-op-wizard-"+project, "#sites", snap.Kind, start, runErr), true
}

// dispatchRunNotification is the seam tests replace, so a run can be driven
// without a notifier behind it.
var dispatchRunNotification = dispatchNotification

func notifyRunFinished(r *run) {
	if n, ok := notificationForRun(r.snapshot(), r.Start); ok {
		dispatchRunNotification(n)
	}
}
