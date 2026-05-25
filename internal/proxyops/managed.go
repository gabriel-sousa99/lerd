package proxyops

import (
	"fmt"
	"strings"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/podman"
)

const (
	defaultNodeMajor  = "20"
	managedUnitPrefix = "lerd-proxy-"
)

// ManagedUnitName returns the systemd unit name for a managed proxy.
func ManagedUnitName(name string) string { return managedUnitPrefix + name }

// generateManagedQuadlet renders the .container file content for p. The
// container joins Network=host so the dev server binds the same port the
// nginx vhost proxies to (host.containers.internal:Port).
func generateManagedQuadlet(p config.Proxy) string {
	node := p.NodeVersion
	if node == "" {
		node = defaultNodeMajor
	}
	cmd := strings.ReplaceAll(p.Command, `"`, `\"`)

	var b strings.Builder
	b.WriteString("[Unit]\n")
	fmt.Fprintf(&b, "Description=Lerd proxy dev server (%s)\n", p.Name)
	b.WriteString("After=network-online.target\n\n")

	b.WriteString("[Container]\n")
	fmt.Fprintf(&b, "Image=docker.io/library/node:%s-alpine\n", node)
	fmt.Fprintf(&b, "ContainerName=%s\n", ManagedUnitName(p.Name))
	b.WriteString("Network=host\n")
	b.WriteString("WorkingDir=/app\n")
	fmt.Fprintf(&b, "Volume=%s:/app:Z\n", p.Path)
	b.WriteString("Environment=HOME=/app\n")
	fmt.Fprintf(&b, "Exec=sh -lc \"%s\"\n\n", cmd)

	b.WriteString("[Service]\n")
	b.WriteString("Restart=on-failure\n")
	b.WriteString("RestartSec=5\n")

	if p.AutoStart {
		b.WriteString("\n[Install]\nWantedBy=default.target\n")
	}
	return b.String()
}

// WriteManagedQuadlet persists the .container file via WriteQuadlet so it
// goes through the same BindForLAN / autostart-disabled pipeline as every
// other lerd unit. The unit becomes available after a daemon-reload.
func WriteManagedQuadlet(p config.Proxy) error {
	if !p.Managed {
		return fmt.Errorf("proxy %s não está em managed mode", p.Name)
	}
	content := generateManagedQuadlet(p)
	if err := podman.WriteQuadlet(ManagedUnitName(p.Name), content); err != nil {
		return err
	}
	return podman.DaemonReloadIfNeeded(true)
}

// RemoveManagedQuadlet stops the unit (best effort) and deletes the file.
func RemoveManagedQuadlet(name string) error {
	unit := ManagedUnitName(name) + ".service"
	_ = podman.StopUnit(unit)
	if err := podman.RemoveQuadlet(ManagedUnitName(name)); err != nil {
		return err
	}
	return podman.DaemonReloadIfNeeded(true)
}

// StartManaged starts the dev-server unit.
func StartManaged(name string) error {
	return podman.StartUnit(ManagedUnitName(name) + ".service")
}

// StopManaged stops the dev-server unit.
func StopManaged(name string) error {
	return podman.StopUnit(ManagedUnitName(name) + ".service")
}
