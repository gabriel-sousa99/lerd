package podman

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// PortOwner names the lerd unit that publishes a host port and the
// container-internal port it maps to, which is what `lerd service port` needs to
// move a secondary mapping.
type PortOwner struct {
	Unit          string
	Service       string
	ContainerPort int
}

// PublishedPortOwner reports which lerd unit publishes hostPort. It reads the
// quadlets rather than the running containers, so a stopped service that will
// claim the port at its next boot counts too, which is exactly the case a port
// conflict has to be caught in.
func PublishedPortOwner(hostPort int) (PortOwner, bool) {
	if hostPort <= 0 {
		return PortOwner{}, false
	}
	entries, err := filepath.Glob(filepath.Join(config.QuadletDir(), "lerd-*.container"))
	if err != nil {
		return PortOwner{}, false
	}
	for _, path := range entries {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			spec, ok := strings.CutPrefix(strings.TrimSpace(line), "PublishPort=")
			if !ok {
				continue
			}
			// The IPv6 form of the same mapping is published alongside the IPv4
			// one and parses to 0 here, so matching either is enough.
			if PrimaryHostPort([]string{spec}) != hostPort {
				continue
			}
			unit := strings.TrimSuffix(filepath.Base(path), ".container")
			return PortOwner{
				Unit:          unit,
				Service:       strings.TrimPrefix(unit, "lerd-"),
				ContainerPort: ContainerPort(spec),
			}, true
		}
	}
	return PortOwner{}, false
}
