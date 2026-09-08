package cli

import (
	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/feedback"
	lerdSystemd "github.com/gabriel-sousa99/lerd/internal/systemd"
)

// RemoveStaleWorkerUnits tears down every unit file left behind for a worker the
// site no longer declares, and returns how many it removed. The doctor's fix for
// the stale_workers finding, kept here rather than in sitedoctor so it goes
// through the same teardown as `lerd worker stop`: disable, stop, drop the timer
// sibling and the exec-mode artifacts, daemon-reload.
//
// Resolves the declared set again rather than trusting the report it came from,
// because a store fetch between the check and the fix can put the worker back.
func RemoveStaleWorkerUnits(site config.Site) (int, error) {
	fw, _ := config.GetFrameworkForDir(site.Framework, site.Path)
	declared, ok := config.DeclaredWorkerNames(site, fw)
	if !ok {
		return 0, nil
	}
	removed := 0
	var firstErr error
	for _, wu := range lerdSystemd.StaleWorkerUnits(site.Name, declared) {
		if err := stopWorkerUnit(wu.Unit, wu.Worker, site.Name+"/"+wu.Worker); err != nil {
			feedback.Warn("removing stale worker unit %s: %v", wu.Unit, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		removed++
	}
	return removed, firstErr
}
