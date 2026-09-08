package ui

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestGraphicalEnvPreservesBaseEnvAndPatchesDisplay(t *testing.T) {
	t.Setenv("LERD_TEST_SENTINEL", "abc123")
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("DISPLAY", "")

	runtimeDir := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	if err := os.WriteFile(runtimeDir+"/wayland-7", []byte{}, 0o600); err != nil {
		t.Fatalf("seed wayland socket: %v", err)
	}

	env := graphicalEnv()

	var sawSentinel, sawWayland, sawRuntimeDir bool
	var waylandVal string
	for _, kv := range env {
		switch {
		case kv == "LERD_TEST_SENTINEL=abc123":
			sawSentinel = true
		case strings.HasPrefix(kv, "WAYLAND_DISPLAY="):
			sawWayland = true
			waylandVal = strings.TrimPrefix(kv, "WAYLAND_DISPLAY=")
		case kv == "XDG_RUNTIME_DIR="+runtimeDir:
			sawRuntimeDir = true
		}
	}

	if !sawSentinel {
		t.Error("graphicalEnv dropped base environment entry")
	}
	if !sawRuntimeDir {
		t.Error("graphicalEnv did not preserve XDG_RUNTIME_DIR")
	}
	if !sawWayland {
		t.Error("graphicalEnv did not probe WAYLAND_DISPLAY from XDG_RUNTIME_DIR")
	}
	if sawWayland && waylandVal != "wayland-7" {
		t.Errorf("WAYLAND_DISPLAY = %q, want wayland-7", waylandVal)
	}
}

func TestTerminalDirCandidatesOpenPtyxisInDir(t *testing.T) {
	t.Setenv("TERMINAL", "")
	const dir = "/home/user/project"
	var ptyxis *terminalCmd
	for _, c := range terminalDirCandidates(dir) {
		if c.bin == "ptyxis" {
			cp := c
			ptyxis = &cp
		}
	}
	if ptyxis == nil {
		t.Fatal("ptyxis is not among the terminal candidates")
	}
	joined := strings.Join(ptyxis.args, " ")
	// ptyxis is single-instance: without --new-window (or --tab/-x) it ignores
	// --working-directory and opens a new window in $HOME instead of the site.
	if !strings.Contains(joined, "--new-window") {
		t.Errorf("ptyxis args %v lack --new-window, so %q would be ignored", ptyxis.args, dir)
	}
	if !strings.Contains(joined, dir) {
		t.Errorf("ptyxis args %v do not carry the target dir %q", ptyxis.args, dir)
	}
}

func TestGraphicalEnvDoesNotDuplicateKeys(t *testing.T) {
	t.Setenv("XDG_SESSION_TYPE", "wayland")
	env := graphicalEnv()
	count := 0
	for _, kv := range env {
		if strings.HasPrefix(kv, "XDG_SESSION_TYPE=") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("XDG_SESSION_TYPE appears %d times, want 1", count)
	}
}

// The terminal the user picked in System Settings is the one to open, so it has
// to outrank every emulator that merely happens to be installed. Only $TERMINAL,
// an explicit override, comes first.
func TestMacDefaultTerminalBundle_ReadsTheShellScriptHandler(t *testing.T) {
	const plist = `{"LSHandlers":[
		{"LSHandlerRoleAll":"com.apple.ical","LSHandlerURLScheme":"webcal"},
		{"LSHandlerContentType":"com.apple.terminal.shell-script","LSHandlerRoleAll":"dev.warp.warp-stable"}
	]}`
	if got := macDefaultTerminalBundle([]byte(plist)); got != "dev.warp.warp-stable" {
		t.Errorf("bundle = %q, want dev.warp.warp-stable", got)
	}
}

// Setting the default from Finder's Get Info panel writes the shell role rather
// than the catch-all one, and that choice counts just the same.
func TestMacDefaultTerminalBundle_FallsBackToTheShellRole(t *testing.T) {
	const plist = `{"LSHandlers":[
		{"LSHandlerContentType":"com.apple.terminal.shell-script","LSHandlerRoleShell":"com.googlecode.iterm2"}
	]}`
	if got := macDefaultTerminalBundle([]byte(plist)); got != "com.googlecode.iterm2" {
		t.Errorf("bundle = %q, want com.googlecode.iterm2", got)
	}
}

// A user who never changed their terminal has no entry at all, and unreadable
// input must not be mistaken for one either. Both leave the caller on its
// ordinary fallbacks rather than opening nothing.
func TestMacDefaultTerminalBundle_EmptyWhenUnset(t *testing.T) {
	for name, plist := range map[string]string{
		"no handlers":    `{"LSHandlers":[]}`,
		"other handlers": `{"LSHandlers":[{"LSHandlerContentType":"public.html","LSHandlerRoleAll":"com.apple.safari"}]}`,
		"not a plist":    `<?xml version="1.0"?>`,
		"empty document": ``,
	} {
		if got := macDefaultTerminalBundle([]byte(plist)); got != "" {
			t.Errorf("%s: bundle = %q, want empty", name, got)
		}
	}
}

// Warp takes a directory argument through open, the same as iTerm and Terminal,
// so it needs no special-casing beyond being detected and offered.
func TestTerminalDirCandidates_OffersWarpBeforeTerminal(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the macOS terminals are only offered on darwin")
	}
	if _, err := os.Stat("/Applications/Warp.app"); err != nil {
		t.Skip("Warp is not installed")
	}
	t.Setenv("TERMINAL", "")

	const dir = "/Users/me/project"
	warp, terminal := -1, -1
	for i, c := range terminalDirCandidates(dir) {
		joined := strings.Join(c.args, " ")
		switch {
		case strings.Contains(joined, "-a Warp"):
			warp = i
			if !strings.Contains(joined, dir) {
				t.Errorf("warp args %v do not carry the target dir %q", c.args, dir)
			}
		case strings.Contains(joined, "-a Terminal"):
			terminal = i
		}
	}
	if warp == -1 {
		t.Fatal("Warp is not among the terminal candidates")
	}
	if terminal != -1 && warp > terminal {
		t.Errorf("Warp at %d is offered after Apple Terminal at %d", warp, terminal)
	}
}
