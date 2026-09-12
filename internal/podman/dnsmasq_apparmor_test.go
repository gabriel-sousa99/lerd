package podman

import (
	"strings"
	"testing"
)

// quadletExec returns the Exec= line of the embedded lerd-dns quadlet.
func quadletExec(t *testing.T) string {
	t.Helper()
	content, err := GetQuadletTemplate("lerd-dns.container")
	if err != nil {
		t.Fatalf("GetQuadletTemplate() = %v", err)
	}
	for _, line := range strings.Split(content, "\n") {
		if v, ok := strings.CutPrefix(line, "Exec="); ok {
			return v
		}
	}
	t.Fatalf("no Exec= line in the quadlet\n\n%s", content)
	return ""
}

// AppArmor attaches profiles by executable path and does not know the binary is
// inside a container, so shipping it at /usr/{bin,sbin}/dnsmasq picks up the
// host's dnsmasq profile, which denies podman's SIGTERM and SIGKILL.
func TestDNSMasqContainerfile_KeepsTheBinaryOffTheHostProfilePaths(t *testing.T) {
	if DNSMasqExec == "dnsmasq" {
		t.Fatal("the binary must not keep the name the host profile attaches to")
	}
	for _, path := range []string{"/usr/bin/" + DNSMasqExec, "/usr/sbin/" + DNSMasqExec} {
		if strings.Contains(dnsmasqContainerfile, path) {
			t.Errorf("image installs the executable at %s, which stock AppArmor profiles claim", path)
		}
	}
	if !strings.Contains(dnsmasqContainerfile, "/usr/local/bin/"+DNSMasqExec) {
		t.Errorf("image does not place the executable at /usr/local/bin/%s\n\n%s", DNSMasqExec, dnsmasqContainerfile)
	}
}

// The quadlet and the Containerfile are edited in different files, so a rename
// on one side alone leaves the unit starting a binary the image does not carry.
func TestDNSMasqQuadlet_ExecsTheBinaryTheImageBuilds(t *testing.T) {
	exec := quadletExec(t)
	if name, _, _ := strings.Cut(exec, " "); name != DNSMasqExec {
		t.Errorf("quadlet execs %q, but the image builds %q", name, DNSMasqExec)
	}
}

// An install that keeps a stale image never rebuilds, so the tag has to move
// whenever the image's contents stop matching what the quadlet asks for.
func TestDNSMasqQuadlet_RunsTheTagTheBuildProduces(t *testing.T) {
	content, err := GetQuadletTemplate("lerd-dns.container")
	if err != nil {
		t.Fatalf("GetQuadletTemplate() = %v", err)
	}
	if !strings.Contains(content, "Image="+DNSMasqImage+"\n") {
		t.Errorf("quadlet does not run Image=%s\n\n%s", DNSMasqImage, content)
	}
}

// A stop that outruns the 60s user@ shutdown window takes the whole session
// down with it, so the unit needs its own bound. StopTimeout= stays out: podman
// <5.0 aborts on the key and leaves the install with no service units (#299).
func TestDNSMasqQuadlet_BoundsTheStopBelowTheShutdownWindow(t *testing.T) {
	content, err := GetQuadletTemplate("lerd-dns.container")
	if err != nil {
		t.Fatalf("GetQuadletTemplate() = %v", err)
	}
	var bounded, podmanKey bool
	for _, line := range strings.Split(content, "\n") {
		bounded = bounded || strings.HasPrefix(line, "TimeoutStopSec=")
		podmanKey = podmanKey || strings.HasPrefix(line, "StopTimeout=")
	}
	if !bounded {
		t.Errorf("quadlet leaves the stop unbounded\n\n%s", content)
	}
	if podmanKey {
		t.Error("the StopTimeout= key must stay out of quadlets podman <5.0 reads")
	}
}
