package config

import (
	"os"
	"path/filepath"
	"testing"
)

// composerProject writes a project with the manifest and, when body is not
// empty, the lock composer would have resolved from it.
func composerProject(t *testing.T, manifest, lock string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if lock != "" {
		writeLock(t, dir, lock)
	}
	return dir
}

// writeLock gives a project the lock composer would have written for it.
func writeLock(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "composer.lock"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

// A framework that stands in for its split packages is the only entry composer
// writes, so the replaced name exists nowhere but under its replace map. This is
// the Tempest case: tempest/app requires tempest/framework alone.
func TestComposerHasInstalled_findsAReplacedPackage(t *testing.T) {
	dir := composerProject(t,
		`{"require": {"tempest/framework": "^3.0"}}`,
		`{"packages": [{"name": "tempest/framework", "version": "v3.19.0",
		  "replace": {"tempest/database": "self.version"}}]}`)

	if !ComposerHasInstalled(dir, "tempest/database") {
		t.Error("a replaced package is installed and must count as present")
	}
	if ComposerHasPackage(dir, "tempest/database") {
		t.Error("the manifest lookup must stay manifest-only")
	}
}

// A tool that arrived under a distribution rather than being asked for by name
// is still there to run.
func TestComposerHasInstalled_findsATransitiveDependency(t *testing.T) {
	dir := composerProject(t,
		`{"require": {"acme/distribution": "^1.0"}}`,
		`{"packages": [{"name": "acme/distribution", "version": "1.2.0"}],
		  "packages-dev": [{"name": "drush/drush", "version": "13.7.1"}]}`)

	if !ComposerHasInstalled(dir, "drush/drush") {
		t.Error("a package the lock installed must count as present")
	}
	if ComposerHasInstalled(dir, "acme/absent") {
		t.Error("a package nothing installed must not count as present")
	}
}

// A project that has never been installed has only its manifest to answer with.
func TestComposerHasInstalled_withoutALockAnswersFromTheManifest(t *testing.T) {
	dir := composerProject(t, `{"require": {"laravel/horizon": "^5.0"}}`, "")

	if !ComposerHasInstalled(dir, "laravel/horizon") {
		t.Error("a declared package must count with no lock on disk")
	}
	if ComposerHasInstalled(dir, "laravel/reverb") {
		t.Error("a package neither file names must not count")
	}
}

// Detection asks what the project is, and a library repo testing against Laravel
// has laravel/framework in its lock without being a Laravel site.
func TestMatchesDetectRule_readsTheManifestAlone(t *testing.T) {
	dir := composerProject(t,
		`{"require-dev": {"orchestra/testbench": "^9.0"}}`,
		`{"packages-dev": [{"name": "laravel/framework", "version": "v11.9.0"},
		  {"name": "orchestra/testbench", "version": "v9.0.0"}]}`)

	rule := FrameworkRule{Composer: "laravel/framework"}
	if MatchesDetectRule(dir, rule) {
		t.Error("a framework in the lock alone must not detect the project as that framework")
	}
	if !MatchesRule(dir, rule) {
		t.Error("a check asks what is installed, and it is")
	}
}
