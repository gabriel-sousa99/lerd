package proxyops

import (
	"fmt"
	"os"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/podman"
)

// UpdateOptions carries the fields an edit may change. Nil pointers mean
// "leave unchanged", so the API can do partial updates. Domain and
// Managed are intentionally NOT here: renaming a domain requires
// reissuing cert + DNS work and is better handled via rm+add; flipping
// managed on/off requires quadlet install/uninstall and is also out of
// scope for the edit path.
type UpdateOptions struct {
	Port         *int
	Path         *string
	Command      *string
	NodeVersion  *string
	UpstreamHost *string
	AutoStart    *bool
}

// Test hook so unit tests can avoid hitting systemd. Production wires to
// podman.RestartUnit; tests can swap it for a no-op via StubForTests.
var restartUnitFn = podman.RestartUnit

// Update applies opts to the proxy identified by name. It only touches
// the artifacts that actually changed:
//   - registry is always saved
//   - vhost is regenerated only if Port or UpstreamHost changed
//   - managed quadlet is rewritten + restarted only if Managed=true and a
//     runtime-relevant field (cmd, node, path, autostart) changed AND the
//     unit is currently active.
//
// Returns the updated proxy (with the same Name; Domains is preserved).
func Update(name string, opts UpdateOptions) (*config.Proxy, error) {
	existing, err := config.FindProxy(name)
	if err != nil {
		return nil, err
	}
	updated := *existing // shallow copy is fine — Domains slice is not mutated here

	vhostDirty := false
	managedRuntimeDirty := false

	if opts.Port != nil && *opts.Port != updated.UpstreamPort {
		if *opts.Port <= 0 || *opts.Port > 65535 {
			return nil, fmt.Errorf("porta inválida: %d", *opts.Port)
		}
		updated.UpstreamPort = *opts.Port
		vhostDirty = true
	}
	if opts.UpstreamHost != nil && *opts.UpstreamHost != updated.UpstreamHost {
		updated.UpstreamHost = *opts.UpstreamHost
		vhostDirty = true
	}
	if opts.Path != nil && *opts.Path != updated.Path {
		if *opts.Path != "" {
			if _, err := os.Stat(*opts.Path); err != nil {
				return nil, fmt.Errorf("path inválido: %w", err)
			}
		}
		if updated.Managed && *opts.Path == "" {
			return nil, fmt.Errorf("path obrigatório quando managed=true")
		}
		updated.Path = *opts.Path
		managedRuntimeDirty = true
	}
	if opts.Command != nil && *opts.Command != updated.Command {
		if updated.Managed && *opts.Command == "" {
			return nil, fmt.Errorf("cmd obrigatório quando managed=true")
		}
		updated.Command = *opts.Command
		managedRuntimeDirty = true
	}
	if opts.NodeVersion != nil && *opts.NodeVersion != updated.NodeVersion {
		updated.NodeVersion = *opts.NodeVersion
		managedRuntimeDirty = true
	}
	if opts.AutoStart != nil && *opts.AutoStart != updated.AutoStart {
		updated.AutoStart = *opts.AutoStart
		managedRuntimeDirty = true
	}

	if err := config.AddProxy(updated); err != nil {
		return nil, fmt.Errorf("salvando registry: %w", err)
	}

	if vhostDirty && !updated.Paused {
		if err := RegenerateProxyVhost(updated); err != nil {
			return nil, fmt.Errorf("gerando vhost: %w", err)
		}
		_ = nginxReloadFn()
	}

	if updated.Managed && managedRuntimeDirty {
		// Rewrite the quadlet file with the new fields. If the unit is
		// currently active, bounce it so the change takes effect; if it
		// isn't, the next StartManaged call picks up the new file.
		if err := WriteManagedQuadlet(updated); err != nil {
			return nil, fmt.Errorf("regerando quadlet: %w", err)
		}
		unit := ManagedUnitName(updated.Name) + ".service"
		if state, _ := podman.UnitStatus(unit); state == "active" || state == "activating" {
			if err := restartUnitFn(unit); err != nil {
				return nil, fmt.Errorf("reiniciando %s: %w", unit, err)
			}
		}
	}

	return &updated, nil
}
