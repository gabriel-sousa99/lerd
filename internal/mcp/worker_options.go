package mcp

import (
	"fmt"
	"strings"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// A worker's tunable values come from the tune_command its framework definition
// declares, which is the same place the CLI reads its per-placeholder flags
// from. Nothing here knows what a queue or a timeout is, so a definition that
// makes another worker tunable reaches an assistant without a change in Go.

// siteWorkers resolves the workers declared for a site: from its framework, or
// from the custom_workers a container site's .lerd.yaml declares when it has no
// framework at all.
func siteWorkers(site *config.Site, dir string) map[string]config.FrameworkWorker {
	if site.IsCustomContainer() && site.Framework == "" {
		proj, _ := config.LoadProjectConfig(dir)
		if proj == nil {
			return nil
		}
		return proj.CustomWorkers
	}
	fw, ok := config.GetFrameworkForDir(site.Framework, dir)
	if !ok {
		return nil
	}
	return fw.Workers
}

// workerTuneArgs turns the options a caller passed into the flags the worker's
// generated start command takes. A name the worker does not declare is refused
// rather than dropped on the way to a start that would look like it applied.
func workerTuneArgs(w config.FrameworkWorker, workerName string, options []string) ([]string, error) {
	flags := config.WorkerTuneFlags(w)
	declared := make(map[string]bool, len(flags))
	names := make([]string, len(flags))
	for i, f := range flags {
		declared[f.Name] = true
		names[i] = f.Name
	}
	if len(flags) == 0 {
		return nil, fmt.Errorf("worker %q declares no tunable values; worker(action: \"list\") reports the ones that do", workerName)
	}

	args := make([]string, 0, len(options))
	for _, opt := range options {
		name, value, ok := strings.Cut(opt, "=")
		name = strings.TrimSpace(name)
		if !ok || name == "" {
			return nil, fmt.Errorf("option %q must be written name=value", opt)
		}
		if !declared[name] {
			return nil, fmt.Errorf("worker %q has no tunable value %q; it declares %s", workerName, name, strings.Join(names, ", "))
		}
		// The value is interpolated into the command the worker's unit runs, so
		// whitespace or a newline could add an argument or a systemd directive.
		if strings.ContainsAny(value, " \t\r\n") {
			return nil, fmt.Errorf("invalid %s value: must not contain whitespace", name)
		}
		args = append(args, "--"+name+"="+value)
	}
	return args, nil
}
