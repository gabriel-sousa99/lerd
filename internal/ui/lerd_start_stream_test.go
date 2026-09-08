package ui

import (
	"bufio"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/cli"
)

var errPodmanDown = errors.New("podman machine is not running")

func stubRunStart(t *testing.T, fn func(func(cli.StartEvent), ...string) error) {
	t.Helper()
	orig := runStart
	t.Cleanup(func() { runStart = orig })
	runStart = fn
}

func postStart(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/lerd/start", nil)
	rec := httptest.NewRecorder()
	handleLerdStart(rec, req)
	return rec
}

func streamEvents(t *testing.T, body string) []cli.StartEvent {
	t.Helper()
	var out []cli.StartEvent
	sc := bufio.NewScanner(strings.NewReader(body))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var evt cli.StartEvent
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			t.Fatalf("line %q is not a StartEvent: %v", line, err)
		}
		out = append(out, evt)
	}
	return out
}

// Starting a unit boots it out of launchd first, which is a SIGTERM. Asking the
// start to restart lerd-ui therefore kills the process running it, leaving the
// dashboard down behind a 502 from nginx.
func TestDashboardStartNeverRestartsTheUIItRunsIn(t *testing.T) {
	var gotSkip []string
	stubRunStart(t, func(_ func(cli.StartEvent), skip ...string) error {
		gotSkip = skip
		return nil
	})

	postStart(t)

	if !slices.Contains(gotSkip, "lerd-ui") {
		t.Errorf("start skip = %v, want it to contain lerd-ui", gotSkip)
	}
}

func TestStartStreamsEveryEventAsOneJSONLine(t *testing.T) {
	stubRunStart(t, func(emit func(cli.StartEvent), _ ...string) error {
		emit(cli.StartEvent{Phase: "step", Step: "units", Total: 2})
		emit(cli.StartEvent{Phase: "unit", Unit: "nginx"})
		emit(cli.StartEvent{Phase: "unit", Unit: "mysql", Error: "boom"})
		emit(cli.StartEvent{Phase: "done"})
		return nil
	})

	rec := postStart(t)
	if ct := rec.Header().Get("Content-Type"); ct != "application/x-ndjson" {
		t.Errorf("Content-Type = %q, want application/x-ndjson", ct)
	}

	events := streamEvents(t, rec.Body.String())
	if len(events) != 4 {
		t.Fatalf("got %d events, want 4: %q", len(events), rec.Body.String())
	}
	if events[0].Total != 2 {
		t.Errorf("first event total = %d, want 2", events[0].Total)
	}
	if events[2].Error != "boom" {
		t.Errorf("failed unit lost its error: %+v", events[2])
	}
	if events[3].Phase != "done" {
		t.Errorf("last phase = %q, want done", events[3].Phase)
	}
}

// The stream is already open by the time the sequence errors, so the failure
// has to arrive as an event rather than as a status code.
func TestStartReportsAFailureOnTheStream(t *testing.T) {
	stubRunStart(t, func(emit func(cli.StartEvent), _ ...string) error {
		emit(cli.StartEvent{Phase: "step", Step: "preparing"})
		return errPodmanDown
	})

	events := streamEvents(t, postStart(t).Body.String())
	last := events[len(events)-1]
	if last.Phase != "failed" || last.Error != errPodmanDown.Error() {
		t.Errorf("last event = %+v, want a failed phase carrying the error", last)
	}
}

func TestStartRejectsNonPOST(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/lerd/start", nil)
	rec := httptest.NewRecorder()
	handleLerdStart(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET should be rejected, got %d", rec.Code)
	}
}
