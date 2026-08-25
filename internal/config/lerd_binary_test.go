package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Homebrew installs into a version-pinned keg and repoints opt/<formula> at it
// on every upgrade. A file that records the keg path stops working the moment
// the next `brew upgrade lerd` removes that version, so the opt spelling is the
// one units and shims have to carry.
func TestBrewOptPathRewritesKegPath(t *testing.T) {
	prefix := resolvedTempDir(t)
	keg := filepath.Join(prefix, "Cellar", "lerd", "1.32.0", "bin", "lerd")
	opt := filepath.Join(prefix, "opt", "lerd", "bin", "lerd")
	writeExecutable(t, keg)
	writeExecutable(t, opt)

	if got := brewOptPath(keg); got != opt {
		t.Errorf("brewOptPath(%q) = %q; want the version-independent %q", keg, got, opt)
	}
}

// Without the opt link there is nothing stable to point at, so the keg path is
// still better than a path that does not exist at all.
func TestBrewOptPathKeepsKegWhenOptMissing(t *testing.T) {
	prefix := resolvedTempDir(t)
	keg := filepath.Join(prefix, "Cellar", "lerd", "1.32.0", "bin", "lerd")
	writeExecutable(t, keg)

	if got := brewOptPath(keg); got != "" {
		t.Errorf("brewOptPath(%q) = %q; want %q so the caller keeps the real path", keg, got, "")
	}
}

func TestBrewOptPathIgnoresOrdinaryInstalls(t *testing.T) {
	for _, path := range []string{
		"/home/u/.local/bin/lerd",
		"/usr/bin/lerd",
		"/nix/store/abc123-lerd-1.32.0/bin/lerd",
	} {
		if got := brewOptPath(path); got != "" {
			t.Errorf("brewOptPath(%q) = %q; want %q", path, got, "")
		}
	}
}

// The ordinary install resolves symlinks, so a unit written today survives the
// ~/.local/bin symlink being repointed or removed later.
func TestLerdBinaryResolvesSymlink(t *testing.T) {
	dir := resolvedTempDir(t)
	real := filepath.Join(dir, "versions", "lerd")
	link := filepath.Join(dir, "lerd")
	writeExecutable(t, real)
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}

	prev := selfExecutable
	selfExecutable = func() (string, error) { return link, nil }
	defer func() { selfExecutable = prev }()

	if got := LerdBinary(); got != real {
		t.Errorf("LerdBinary() = %q; want the resolved %q", got, real)
	}
}

func TestLerdBinaryPrefersBrewOptPath(t *testing.T) {
	prefix := resolvedTempDir(t)
	keg := filepath.Join(prefix, "Cellar", "lerd", "1.32.0", "bin", "lerd")
	opt := filepath.Join(prefix, "opt", "lerd", "bin", "lerd")
	writeExecutable(t, keg)
	writeExecutable(t, opt)

	prev := selfExecutable
	selfExecutable = func() (string, error) { return keg, nil }
	defer func() { selfExecutable = prev }()

	if got := LerdBinary(); got != opt {
		t.Errorf("LerdBinary() = %q; want %q, which brew repoints on upgrade", got, opt)
	}
}

// The keg brew is replacing is still on disk when its post-install hook runs;
// the cleanup that deletes it comes later, with nobody watching. So an older
// keg of the same formula counts as superseded while it is still there.
func TestSupersededBinaryCoversAnOlderKegOfTheSameFormula(t *testing.T) {
	prefix := resolvedTempDir(t)
	previous := filepath.Join(prefix, "Cellar", "lerd", "1.32.0", "bin", "lerd")
	opt := filepath.Join(prefix, "opt", "lerd", "bin", "lerd")
	writeExecutable(t, previous)
	writeExecutable(t, opt)

	if !SupersededBinary(previous, opt) {
		t.Errorf("SupersededBinary(%q, %q) = false; want the older keg superseded", previous, opt)
	}
}

func TestSupersededBinaryCoversABinaryThatIsGone(t *testing.T) {
	if !SupersededBinary("/home/linuxbrew/.linuxbrew/Cellar/lerd/1.31.0/bin/lerd", "/usr/bin/lerd") {
		t.Error("a recorded binary that is not on disk should count as superseded")
	}
}

// A working install of a different provenance keeps its units and shims: a brew
// install has no claim on the lerd someone installed with the script.
func TestSupersededBinaryLeavesAnotherInstallAlone(t *testing.T) {
	prefix := resolvedTempDir(t)
	script := filepath.Join(prefix, "home", ".local", "bin", "lerd")
	opt := filepath.Join(prefix, "opt", "lerd", "bin", "lerd")
	writeExecutable(t, script)
	writeExecutable(t, opt)

	if SupersededBinary(script, opt) {
		t.Errorf("SupersededBinary(%q, %q) = true; want another install left alone", script, opt)
	}
	if SupersededBinary(opt, opt) {
		t.Error("a binary that is the one running is not superseded by itself")
	}
	if SupersededBinary("", opt) || SupersededBinary(script, "") {
		t.Error("an unknown path on either side is not something to repoint")
	}
}

// resolvedTempDir is t.TempDir() with symlinks resolved, so a comparison
// against a path LerdBinary resolved does not fail on macOS, where the temp dir
// lives under the /var → /private/var symlink.
func resolvedTempDir(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
}
