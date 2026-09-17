package cli

import (
	"fmt"
	"io"

	"github.com/gabriel-sousa99/lerd/internal/podman"
)

// warnStaleMountRepairs reports the bind mounts a preflight dropped from the
// quadlets, naming the site a path belonged to when it maps to one. Shared by
// `lerd start` (stdout, inside its startup report) and the on-demand FPM start
// behind the php/artisan shims (stderr, so the shim's stdout stays clean).
func warnStaleMountRepairs(w io.Writer, repairs []podman.MountRepair) {
	for _, r := range repairs {
		if r.Site != "" {
			fmt.Fprintf(w, "  WARN: %s no longer exists (site %s), removed from %s\n", r.Path, r.Site, r.Unit)
			continue
		}
		fmt.Fprintf(w, "  WARN: %s no longer exists, removed from %s\n", r.Path, r.Unit)
	}
}
