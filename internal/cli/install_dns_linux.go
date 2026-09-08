//go:build linux

package cli

import (
	"io"

	"github.com/gabriel-sousa99/lerd/internal/dns"
	"github.com/gabriel-sousa99/lerd/internal/feedback"
	"github.com/gabriel-sousa99/lerd/internal/imagepull"
	"github.com/gabriel-sousa99/lerd/internal/podman"
	"github.com/gabriel-sousa99/lerd/internal/services"
)

// writeDNSUnit writes the container quadlet for the dnsmasq DNS service on Linux.
func writeDNSUnit(_ io.Writer) error {
	content, err := podman.GetQuadletTemplate("lerd-dns.container")
	if err != nil {
		return err
	}
	return services.Mgr.WriteContainerUnit("lerd-dns", content)
}

// ensureDNSImageForStart ensures the lerd-dnsmasq container image exists on Linux.
func ensureDNSImageForStart() {
	// Ignore errors — the image will be built again during RunParallel if missing.
	if !podman.ImageExists(podman.DNSMasqImage) {
		_ = podman.BuildDNSMasqImage(io.Discard, dns.ReadUpstreamDNS())
	}
}

// dnsImagePlan discloses what pullDNSImages downloads. The dnsmasq build adds
// only apk packages on top of the base, so the base pull is the whole of it.
func dnsImagePlan() imagepull.Plan {
	return imagepull.Plan{
		imagepull.Pull(podman.DNSMasqBaseImage, "base for the lerd-dns image"),
	}
}

// pullDNSImages returns build jobs to pull alpine and build the dnsmasq container image.
func pullDNSImages() []BuildJob {
	return []BuildJob{
		{
			Label: "Pulling alpine:latest",
			Run: func(w io.Writer) error {
				return podman.PullImageTo(podman.DNSMasqBaseImage, w)
			},
		},
		{
			Label: "Building dnsmasq image",
			Run: func(w io.Writer) error {
				return podman.BuildDNSMasqImage(w, dns.ReadUpstreamDNS())
			},
		},
	}
}

// isDNSContainerUnit returns true on Linux since DNS uses a Podman container.
func isDNSContainerUnit() bool { return true }

// ensureDNSServiceUpdated is a no-op on Linux — DNS always uses a container.
func ensureDNSServiceUpdated(_ io.Writer) error { return nil }

// removeDNSContainerIfRunning is a no-op on Linux.
func removeDNSContainerIfRunning() {}

// nativeDNSRestart is a no-op on Linux — DNS is a container unit managed by systemd.
func nativeDNSRestart() error { return nil }

// needsDNSServiceInstall always returns false on Linux (container quadlet handles it).
func needsDNSServiceInstall() bool { return false }

// teardownDNS stops the lerd-dns container, removes its quadlet, and reloads
// the user manager so a subsequent `lerd install` does not silently restart
// the unit. Called from runInstall when the user flips dns.enabled from true
// to false; safe to call when nothing is installed.
func teardownDNS() {
	_ = services.Mgr.Stop("lerd-dns")
	_ = services.Mgr.RemoveContainerUnit("lerd-dns")
	_ = services.Mgr.DaemonReload()

	// Only when lerd actually wrote resolver config. install.go calls this on
	// every run where DNS is off, not just on a true->false flip, so an
	// unconditional teardown would revert interfaces and restart NetworkManager on
	// every `lerd install` for someone who never let lerd near their resolver.
	if !dnsResolverConfigured() {
		return
	}
	// Announced with the lock glyph: the removals run as root. They are granted in
	// the sudoers drop-in so they do not prompt, but the header keeps the teardown
	// visible in the output.
	feedback.Sudo("Removing DNS configuration")
	dnsTeardown()
}

// Seams so tests can drive the disable path without shelling out to sudo or
// depending on what the test host happens to have installed.
var (
	dnsTeardown           = dns.Teardown
	dnsResolverConfigured = dns.ResolverConfigured
)
