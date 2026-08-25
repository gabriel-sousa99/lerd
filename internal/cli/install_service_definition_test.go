package cli

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// A site keeps a copy of the service definition it was linked with. When that
// copy has gone stale against the store, install must regenerate the quadlet
// from the installed definition instead, or the service goes back to an old
// image and an old host port that another service now publishes.
func TestInstalledServiceDefinition_prefersTheInstalledYAML(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	installed := &config.CustomService{
		Name:  "adminui",
		Image: "docker.io/library/adminui:latest",
		Ports: []string{"8080:80"},
	}
	if err := config.SaveCustomService(installed); err != nil {
		t.Fatalf("saving the installed definition: %v", err)
	}

	stale := &config.CustomService{
		Name:  "adminui",
		Image: "docker.io/adminui/adminui:latest",
		Ports: []string{"8081:80"},
	}

	got := installedServiceDefinition("adminui", stale)
	if got.Image != installed.Image {
		t.Errorf("image = %q, want the installed %q", got.Image, installed.Image)
	}
	if len(got.Ports) != 1 || got.Ports[0] != "8080:80" {
		t.Errorf("ports = %v, want the installed [8080:80]", got.Ports)
	}
}

// A service declared inline in .lerd.yaml and never installed globally has no
// other definition, so the site copy stays the one install writes.
func TestInstalledServiceDefinition_fallsBackToTheSiteCopy(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	inline := &config.CustomService{
		Name:  "inlineonly",
		Image: "docker.io/library/inlineonly:latest",
		Ports: []string{"9500:80"},
	}

	got := installedServiceDefinition("inlineonly", inline)
	if got != inline {
		t.Errorf("got %+v, want the inline definition", got)
	}
}
