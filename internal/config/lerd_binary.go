package config

import (
	"os"
	"path/filepath"
	"strings"
)

// selfExecutable is a seam for tests.
var selfExecutable = os.Executable

// LerdBinary returns the absolute path of the running lerd binary, spelled the
// way anything that outlives this process (a systemd unit, a launchd plist, a
// shim on PATH) has to record it. Symlinks are resolved so a link that moves
// later cannot break the file, with Homebrew as the exception: its kegs are
// version-pinned, so the resolved path names a directory the next
// `brew upgrade lerd` deletes. Falls back to ~/.local/bin/lerd, lerd's own
// install location, when the running executable cannot be resolved.
func LerdBinary() string {
	exe, err := selfExecutable()
	if err != nil || exe == "" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "bin", "lerd")
	}
	if resolved, rErr := filepath.EvalSymlinks(exe); rErr == nil {
		exe = resolved
	}
	if opt := brewOptPath(exe); opt != "" {
		return opt
	}
	return exe
}

// SupersededBinary reports whether recorded names a lerd binary that current
// has taken over from: one that is no longer on disk, or an older Homebrew keg
// of the formula current is the opt link for. A path belonging to some other
// install (a script install under ~/.local/bin, a distro package) is never
// superseded, so a brew install standing next to one cannot claim its units.
func SupersededBinary(recorded, current string) bool {
	if recorded == "" || current == "" || recorded == current {
		return false
	}
	if _, err := os.Stat(recorded); err != nil {
		return true
	}
	return brewOptPath(recorded) == current
}

// brewOptPath maps a Homebrew keg path (<prefix>/Cellar/lerd/1.32.0/bin/lerd) to
// its opt equivalent (<prefix>/opt/lerd/bin/lerd), the link brew repoints at the
// current version on every upgrade. Returns "" for a path outside a Cellar, or
// when the opt link is absent, so the caller keeps the path it already has.
func brewOptPath(path string) string {
	prefix, keg, ok := strings.Cut(path, "/Cellar/")
	if !ok {
		return ""
	}
	parts := strings.SplitN(keg, "/", 3) // formula / version / path within the keg
	if len(parts) != 3 || parts[0] == "" || parts[2] == "" {
		return ""
	}
	opt := filepath.Join(prefix, "opt", parts[0], parts[2])
	if _, err := os.Stat(opt); err != nil {
		return ""
	}
	return opt
}
