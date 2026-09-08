package sitedoctor

import (
	"fmt"
	"strings"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/systemd"
)

// checkStaleWorkers reports unit files left behind for workers the site no
// longer declares. A definition that retires a worker reaches every install
// within a day, and nothing on the way in reconciles the units against it, so
// the unit stays on disk, stays armed for boot, and stays something the
// self-heal detector walks. Nothing else asks the opposite question of what a
// unit still answers to, which is why it is asked here.
//
// Skipped when the framework cannot be resolved, since an install that cannot
// read its store would otherwise call every worker stale.
func checkStaleWorkers(path string, fw *config.Framework) (Check, bool) {
	site, err := config.FindSiteByPath(path)
	if err != nil || site == nil {
		return Check{}, false
	}
	declared, ok := config.DeclaredWorkerNames(*site, fw)
	if !ok {
		return Check{}, false
	}
	stale := systemd.StaleWorkerUnits(site.Name, declared)
	if len(stale) == 0 {
		return Check{Name: "stale_workers", Status: StatusOK, Detail: "every worker unit answers to a definition"}, true
	}
	names := make([]string, 0, len(stale))
	for _, wu := range stale {
		names = append(names, wu.Worker)
	}
	noun := "a worker"
	if len(names) > 1 {
		noun = fmt.Sprintf("%d workers", len(names))
	}
	return Check{
		Name:   "stale_workers",
		Status: StatusWarn,
		Detail: fmt.Sprintf("unit files are still installed for %s this site no longer declares (%s); remove them to stop them starting at boot", noun, strings.Join(names, ", ")),
		Fix:    FixStaleWorkers,
	}, true
}
