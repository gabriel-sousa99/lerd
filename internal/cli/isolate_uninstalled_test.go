package cli

import (
	"strings"
	"testing"
)

// isolate on a directory that is not yet a site writes the pin and nothing else,
// so pinning a version this machine has never installed reported success and the
// next command in that directory failed with "PHP 8.4 is not installed". The pin
// is still correct; what was missing is saying the version has nowhere to run
// yet, and how to get it.
func TestPHPVersionNotInstalledNote_namesTheMissingVersion(t *testing.T) {
	isolateUnitDir(t)

	note := phpVersionNotInstalledNote("8.4")
	if note == "" {
		t.Fatal("pinning an uninstalled version reported nothing")
	}
	if !strings.Contains(note, "8.4") {
		t.Errorf("note %q does not name the version it is about", note)
	}
	if !strings.Contains(note, "php:rebuild 8.4") {
		t.Errorf("note %q does not name a command that installs the version", note)
	}
}

// A version that is installed needs no note; the pin is all there is to say.
func TestPHPVersionNotInstalledNote_silentForAnInstalledVersion(t *testing.T) {
	isolateUnitDir(t)
	stageFPMQuadlet(t, "8.4")

	if note := phpVersionNotInstalledNote("8.4"); note != "" {
		t.Errorf("note for an installed version: %q", note)
	}
}
