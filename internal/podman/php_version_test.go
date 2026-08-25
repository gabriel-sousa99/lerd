package podman

import (
	"os/exec"
	"sync"
	"testing"
)

func resetPHPVerCache() {
	phpVerMu.Lock()
	phpVerCache = map[string]phpVerEntry{}
	phpVerProbes = map[string]*sync.Mutex{}
	phpVerMu.Unlock()
}

// A version's full patch is probed from the image once and cached, keyed on the
// image ID.
func TestRefreshPHPVersion_probesAndCaches(t *testing.T) {
	origID, origExec := imageIDFn, execCommand
	t.Cleanup(func() { imageIDFn = origID; execCommand = origExec; resetPHPVerCache() })
	resetPHPVerCache()

	imageIDFn = func(string) string { return "sha256:aaa" }
	probes := 0
	execCommand = func(name string, arg ...string) *exec.Cmd {
		probes++
		return exec.Command("printf", "8.5.1")
	}

	refreshPHPVersion("8.5")
	phpVerMu.Lock()
	got := phpVerCache["8.5"].patch
	phpVerMu.Unlock()
	if got != "8.5.1" {
		t.Fatalf("cached patch = %q, want 8.5.1", got)
	}

	// A second call with the same image must not re-probe.
	refreshPHPVersion("8.5")
	if probes != 1 {
		t.Errorf("probed %d times, want 1 (cache should serve the unchanged image)", probes)
	}

	// A new image ID (a rebuild) must re-probe.
	imageIDFn = func(string) string { return "sha256:bbb" }
	execCommand = func(name string, arg ...string) *exec.Cmd { return exec.Command("printf", "8.5.2") }
	refreshPHPVersion("8.5")
	phpVerMu.Lock()
	got = phpVerCache["8.5"].patch
	phpVerMu.Unlock()
	if got != "8.5.2" {
		t.Errorf("after rebuild cached patch = %q, want 8.5.2", got)
	}
}

// A base-image update rebuilds the image without touching the containerfile, so
// its hash label is byte identical either side of the build. The patch must
// still be re-probed: only the image ID moves.
func TestRefreshPHPVersion_reprobesWhenOnlyTheBaseChanged(t *testing.T) {
	origID, origLabel, origExec := imageIDFn, imageLabelFn, execCommand
	t.Cleanup(func() {
		imageIDFn, imageLabelFn, execCommand = origID, origLabel, origExec
		resetPHPVerCache()
	})
	resetPHPVerCache()

	imageLabelFn = func(_, _ string) string { return "same-containerfile-hash" }
	imageIDFn = func(string) string { return "sha256:before" }
	execCommand = func(name string, arg ...string) *exec.Cmd { return exec.Command("printf", "8.5.1") }
	refreshPHPVersion("8.5")

	imageIDFn = func(string) string { return "sha256:after" }
	execCommand = func(name string, arg ...string) *exec.Cmd { return exec.Command("printf", "8.5.2") }
	refreshPHPVersion("8.5")

	phpVerMu.Lock()
	got := phpVerCache["8.5"].patch
	phpVerMu.Unlock()
	if got != "8.5.2" {
		t.Errorf("cached patch = %q, want 8.5.2 (an unchanged containerfile must not keep the old patch)", got)
	}
}

// RefreshFPMPHPVersion is the synchronous path a just-finished rebuild uses: the
// new patch is in the cache by the time it returns.
func TestRefreshFPMPHPVersion_fillsCacheBeforeReturning(t *testing.T) {
	origID, origExec := imageIDFn, execCommand
	t.Cleanup(func() { imageIDFn = origID; execCommand = origExec; resetPHPVerCache() })
	resetPHPVerCache()

	imageIDFn = func(string) string { return "sha256:aaa" }
	execCommand = func(name string, arg ...string) *exec.Cmd { return exec.Command("printf", "8.5.3") }

	RefreshFPMPHPVersion("8.5")
	phpVerMu.Lock()
	got := phpVerCache["8.5"].patch
	phpVerMu.Unlock()
	if got != "8.5.3" {
		t.Errorf("cached patch = %q, want 8.5.3", got)
	}
}

// An unbuilt image (no ID) yields no patch and no probe.
func TestRefreshPHPVersion_unbuiltImageIsNoop(t *testing.T) {
	origID, origExec := imageIDFn, execCommand
	t.Cleanup(func() { imageIDFn = origID; execCommand = origExec; resetPHPVerCache() })
	resetPHPVerCache()

	imageIDFn = func(string) string { return "" }
	probed := false
	execCommand = func(name string, arg ...string) *exec.Cmd { probed = true; return exec.Command("true") }

	refreshPHPVersion("8.5")
	if probed {
		t.Error("must not probe an image that is not built")
	}
	phpVerMu.Lock()
	_, ok := phpVerCache["8.5"]
	phpVerMu.Unlock()
	if ok {
		t.Error("must not cache anything for an unbuilt image")
	}
}
