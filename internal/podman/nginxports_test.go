package podman

import (
	"strings"
	"testing"
)

// The bundled template publishes the defaults; a host that needs nginx off
// :80/:443 sets nginx.http_port / https_port and the unit has to follow, or
// the setting is inert and there is no supported way to move nginx (#1544).
func TestApplyNginxPorts(t *testing.T) {
	template, err := GetQuadletTemplate("lerd-nginx.container")
	if err != nil {
		t.Fatalf("reading template: %v", err)
	}

	t.Run("configured ports move the host side only", func(t *testing.T) {
		got := ApplyNginxPorts(template, 10080, 10443)
		for _, want := range []string{"PublishPort=10080:80", "PublishPort=10443:443"} {
			if !strings.Contains(got, want) {
				t.Errorf("missing %q in:\n%s", want, got)
			}
		}
		if strings.Contains(got, "PublishPort=80:80") || strings.Contains(got, "PublishPort=443:443") {
			t.Errorf("default mappings survived:\n%s", got)
		}
	})

	t.Run("the defaults are a no-op", func(t *testing.T) {
		if got := ApplyNginxPorts(template, 80, 443); got != template {
			t.Errorf("template changed when ports are the defaults:\n%s", got)
		}
	})

	t.Run("an unset port leaves its mapping alone", func(t *testing.T) {
		got := ApplyNginxPorts(template, 0, 10443)
		if !strings.Contains(got, "PublishPort=80:80") {
			t.Errorf("http mapping should be untouched:\n%s", got)
		}
		if !strings.Contains(got, "PublishPort=10443:443") {
			t.Errorf("https mapping should move:\n%s", got)
		}
	})

	t.Run("everything else is left as it was", func(t *testing.T) {
		got := ApplyNginxPorts(template, 10080, 10443)
		for _, want := range []string{"ContainerName=lerd-nginx", "Network=lerd", "Restart=always"} {
			if !strings.Contains(got, want) {
				t.Errorf("rewrite dropped %q", want)
			}
		}
		if strings.Count(got, "PublishPort=") != strings.Count(template, "PublishPort=") {
			t.Errorf("port line count changed: %d -> %d",
				strings.Count(template, "PublishPort="), strings.Count(got, "PublishPort="))
		}
	})
}
