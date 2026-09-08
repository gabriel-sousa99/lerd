package cli

import (
	"reflect"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// The default version comes first and every active site adds its own once, so
// install brings up exactly the runtimes something on this machine points at.
func TestFPMVersionsToEnsure_defaultFirstThenEachActiveSiteOnce(t *testing.T) {
	sites := []config.Site{
		{Name: "shop", PHPVersion: "8.4"},
		{Name: "blog", PHPVersion: "8.3"},
		{Name: "api", PHPVersion: "8.4"},
		{Name: "inherits", PHPVersion: ""},
	}

	got := fpmVersionsToEnsure("8.3", sites)

	if want := []string{"8.3", "8.4"}; !reflect.DeepEqual(got, want) {
		t.Errorf("versions = %v, want %v", got, want)
	}
}

// A paused or ignored site is not served, so its version is not worth a build.
func TestFPMVersionsToEnsure_skipsPausedAndIgnoredSites(t *testing.T) {
	sites := []config.Site{
		{Name: "paused", PHPVersion: "8.2", Paused: true},
		{Name: "ignored", PHPVersion: "8.1", Ignored: true},
		{Name: "live", PHPVersion: "8.5"},
	}

	got := fpmVersionsToEnsure("8.4", sites)

	if want := []string{"8.4", "8.5"}; !reflect.DeepEqual(got, want) {
		t.Errorf("versions = %v, want %v", got, want)
	}
}

// No default configured yet (a fresh install) must not queue an empty version.
func TestFPMVersionsToEnsure_dropsEmptyVersions(t *testing.T) {
	got := fpmVersionsToEnsure("", []config.Site{{Name: "orphan", PHPVersion: ""}})

	if len(got) != 0 {
		t.Errorf("versions = %v, want none", got)
	}
}

// Only a version with something to build gets the progress loader. The rest are
// no-ops whose podman output would otherwise land raw in the install log.
func TestFPMEnsurePlan_splitsOnWhatActuallyBuilds(t *testing.T) {
	orig := fpmImageCurrentFn
	t.Cleanup(func() { fpmImageCurrentFn = orig })
	fpmImageCurrentFn = func(v string) bool { return v == "8.3" }

	build, quiet := fpmEnsurePlan([]string{"8.3", "8.4", "8.5"})

	if want := []string{"8.4", "8.5"}; !reflect.DeepEqual(build, want) {
		t.Errorf("build = %v, want %v", build, want)
	}
	if want := []string{"8.3"}; !reflect.DeepEqual(quiet, want) {
		t.Errorf("quiet = %v, want %v", quiet, want)
	}
}
