package ui

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stubContainerRunning swaps the running check so the shell action can be driven
// without a live podman.
func stubContainerRunning(t *testing.T, running bool) {
	t.Helper()
	prev := containerRunning
	containerRunning = func(string) bool { return running }
	t.Cleanup(func() { containerRunning = prev })
}

func postShell(version string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	path := "/api/php-versions/" + version + "/shell"
	handlePHPVersionAction(rec, httptest.NewRequest(http.MethodPost, path, nil))
	return rec
}

func TestPHPShellActionOpensTheVersionContainer(t *testing.T) {
	stubContainerRunning(t, true)
	script := stubOpenTerminal(t, nil)

	rec := postShell("8.4")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"ok":true`) {
		t.Errorf("body = %s, want ok:true", rec.Body.String())
	}
	if !strings.HasSuffix(*script, " shell '8.4'") {
		t.Errorf("script = %q, want it to run lerd's own shell command for 8.4", *script)
	}
	// The container script contains "$PATH"; a hand-built podman exec would let
	// the host shell expand it and shadow the container's own bin directories.
	if strings.Contains(*script, "podman") || strings.Contains(*script, "$PATH") {
		t.Errorf("script = %q, want no shell-expanded podman exec", *script)
	}
}

func TestPHPShellActionRefusesAStoppedVersion(t *testing.T) {
	stubContainerRunning(t, false)
	script := stubOpenTerminal(t, nil)

	rec := postShell("8.4")
	if !strings.Contains(rec.Body.String(), `"ok":false`) {
		t.Errorf("body = %s, want ok:false", rec.Body.String())
	}
	if *script != "" {
		t.Errorf("opened a terminal for a stopped container: %q", *script)
	}
}

func TestPHPShellActionReportsMissingEmulator(t *testing.T) {
	stubContainerRunning(t, true)
	stubOpenTerminal(t, errors.New("no terminal emulator found"))

	rec := postShell("8.4")
	if !strings.Contains(rec.Body.String(), "no terminal emulator found") {
		t.Errorf("body = %s, want the launcher error surfaced", rec.Body.String())
	}
}
