package config

import (
	"os"
	"path/filepath"
	"testing"
)

// A listing describes the layer from the cache alone: what each package
// declares, which file would serve this project, and whether the project
// requires it at all.
func TestListStorePackages_describesTheLayerForAProject(t *testing.T) {
	_, project := packageSandbox(t, []string{
		`{"name":"acme/electron"}`,
		`{"name":"acme/queue"}`,
	}, "acme/electron")
	writePackage(t, "acme-electron", electronPackage)

	got := ListStorePackages(project)
	if len(got) != 2 {
		t.Fatalf("listed %d packages, want 2", len(got))
	}
	// The one this project requires sorts first.
	first := got[0]
	if first.Name != "acme/electron" || !first.Required {
		t.Fatalf("first entry = %+v, want the required acme/electron", first)
	}
	if !first.Cached {
		t.Error("a package with a file on disk must read as cached")
	}
	if len(first.Workers) != 1 || first.Workers[0] != "native" {
		t.Errorf("workers = %v, want [native]", first.Workers)
	}
	if len(first.Commands) != 1 || first.Commands[0] != "native:build" {
		t.Errorf("commands = %v, want [native:build]", first.Commands)
	}
	if first.Doctor != 1 {
		t.Errorf("doctor checks = %d, want 1", first.Doctor)
	}
	// The store publishes the second one, but nothing pulled it and this
	// project does not use it.
	if second := got[1]; second.Cached || second.Required {
		t.Errorf("second entry = %+v, want neither cached nor required", second)
	}
}

// Outside a project nothing is required, and the listing still names what the
// store publishes.
func TestListStorePackages_outsideAProject(t *testing.T) {
	packageSandbox(t, []string{`{"name":"acme/electron"}`})
	writePackage(t, "acme-electron", electronPackage)

	got := ListStorePackages("")
	if len(got) != 1 {
		t.Fatalf("listed %d packages, want 1", len(got))
	}
	if got[0].Required {
		t.Error("no project means nothing is required")
	}
	if !got[0].Cached {
		t.Error("the cached file should still be read")
	}
}

// The file named is the one that would serve the project, not simply the newest
// the store publishes.
func TestListStorePackages_namesTheVersionServingTheProject(t *testing.T) {
	_, project := packageSandbox(t, []string{`{"name":"acme/electron","versions":["5","7"],"latest":"7"}`})
	if err := os.WriteFile(filepath.Join(project, "composer.json"),
		[]byte(`{"require": {"acme/framework": "^11.0", "acme/electron": "^6.0"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	writePackage(t, "acme-electron@5", packageAtVersion("5", "php acme five"))

	got := ListStorePackages(project)
	if len(got) != 1 || got[0].Version != "5" {
		t.Fatalf("version = %q, want the 5 file that serves a 6 install", got[0].Version)
	}
}
