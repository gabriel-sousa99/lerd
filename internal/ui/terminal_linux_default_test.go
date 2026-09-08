package ui

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestKDEDefaultTerminal_readsTheGeneralKey(t *testing.T) {
	body := []byte("[Colors:View]\nTerminalApplication=not-this-one\n\n[General]\nAccentColor=1,2,3\nTerminalApplication=alacritty\n")
	if got := kdeDefaultTerminal(body); got != "alacritty" {
		t.Errorf("kdeDefaultTerminal = %q, want %q", got, "alacritty")
	}
}

// KDE writes the key only once the user changes it, so an untouched install has
// to fall through to the ordinary list rather than name something.
func TestKDEDefaultTerminal_emptyWhenNeverPicked(t *testing.T) {
	body := []byte("[General]\nAccentColor=1,2,3\nBrowserApplication=firefox.desktop\n")
	if got := kdeDefaultTerminal(body); got != "" {
		t.Errorf("kdeDefaultTerminal = %q, want empty", got)
	}
	if got := kdeDefaultTerminal(nil); got != "" {
		t.Errorf("kdeDefaultTerminal(nil) = %q, want empty", got)
	}
}

// A chosen terminal has to be invoked with its own flags. kitty and ghostty take
// the program directly and reject `-e sh -c`, so the generic form would fail and
// drop the user onto whatever happened to be next on PATH.
func TestNamedTerminal_reusesTheFlagsLerdKnows(t *testing.T) {
	got := namedTerminal("kitty", "/srv/app")
	if got.bin != "kitty" || strings.Join(got.args, " ") != "--directory /srv/app" {
		t.Errorf("namedTerminal(kitty) = %+v, want kitty --directory /srv/app", got)
	}

	// An absolute path is what an alternatives link resolves to, and it still
	// has to match the emulator it points at.
	abs := filepath.Join("/usr", "bin", "ghostty")
	if got := namedTerminal(abs, "/srv/app"); got.bin != abs || got.args[0] != "--working-directory=/srv/app" {
		t.Errorf("namedTerminal(%s) = %+v, want its own flag", abs, got)
	}
}

func TestNamedTerminal_fallsBackToTheGenericForm(t *testing.T) {
	got := namedTerminal("some-terminal", "/srv/app")
	if got.bin != "some-terminal" {
		t.Errorf("bin = %q, want some-terminal", got.bin)
	}
	if strings.Join(got.args, " ") != `-e sh -c cd "$0" && exec "$SHELL" /srv/app` {
		t.Errorf("args = %v, want the cd wrapper", got.args)
	}
}

// The chosen terminal outranks the fixed list, or picking one in the desktop's
// settings would change nothing.
func TestTerminalDirCandidates_putsTheChosenTerminalFirst(t *testing.T) {
	t.Setenv("TERMINAL", "my-terminal")
	got := terminalDirCandidates("/srv/app")
	if len(got) == 0 || got[0].bin != "my-terminal" {
		t.Fatalf("first candidate = %+v, want my-terminal", got)
	}
}
