package podman

import (
	"strings"
	"testing"
)

func TestDevServerPublishPorts_Skips56(t *testing.T) {
	if got := devServerPublishPorts("5.6"); got != nil {
		t.Errorf("php 5.6 must not expose dev-server ports, got %v", got)
	}

	want := []string{"0.0.0.0:8000:8000", "0.0.0.0:8001:8001", "0.0.0.0:6001:6001"}
	for _, v := range []string{"7.4", "8.1", "8.3", "8.4"} {
		got := devServerPublishPorts(v)
		if len(got) != len(want) {
			t.Fatalf("version %s: expected %d ports, got %v", v, len(want), got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("version %s port %d = %q, want %q", v, i, got[i], want[i])
			}
		}
	}
}

func TestInjectDevServerPorts_InContainerSection(t *testing.T) {
	tmpl, err := GetQuadletTemplate("lerd-php-fpm.container.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	out := injectDevServerPorts(tmpl, "8.4")
	for _, p := range []string{
		"PublishPort=0.0.0.0:8000:8000",
		"PublishPort=0.0.0.0:8001:8001",
		"PublishPort=0.0.0.0:6001:6001",
	} {
		if !strings.Contains(out, p) {
			t.Errorf("8.4 quadlet missing %q:\n%s", p, out)
		}
	}

	// PublishPort lines must land inside [Container] (before the [Service]
	// section header) or systemd parses them under the wrong section.
	portIdx := strings.Index(out, "PublishPort=0.0.0.0:8000:8000")
	svcIdx := strings.Index(out, "[Service]")
	if portIdx < 0 || svcIdx < 0 || portIdx > svcIdx {
		t.Errorf("PublishPort must appear inside [Container] (before [Service]); port=%d service=%d", portIdx, svcIdx)
	}

	// PHP 5.6 stays clean so it can coexist with a modern version without a
	// host-port collision.
	out56 := injectDevServerPorts(tmpl, "5.6")
	if strings.Contains(out56, "PublishPort=") {
		t.Errorf("php 5.6 quadlet must not contain PublishPort lines:\n%s", out56)
	}
}
