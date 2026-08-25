package linker

import (
	"slices"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// MatchesRegistration reports whether the registry already holds the
// registration this plan would write, so carrying the plan out would change
// nothing. Callers that link only to apply a project's configuration use it to
// skip a link that has no work in it, rather than repeating every provisioning
// step and the summary for a site that is already serving.
//
// Only the fields a link decides are compared. Everything else the registry
// carries is the site's operational state — paused, pinned, grouped, its LAN
// and worktree ports — which no link writes and which must never read as a
// change. A plan that registers nothing has no registration to compare and
// never matches.
func (p *Plan) MatchesRegistration(existing config.Site) bool {
	if !p.Registered() {
		return false
	}
	// An ignored entry is a parked site that was unlinked: it survives only to
	// keep the watcher from re-adding it, and it is hidden from every list and
	// serves nothing. Re-linking has all its work still to do.
	if existing.Ignored {
		return false
	}
	s := p.Site
	return s.Name == existing.Name &&
		config.SamePath(s.Path, existing.Path) &&
		sameDomainSet(s.Domains, existing.Domains) &&
		s.PHPVersion == existing.PHPVersion &&
		s.NodeVersion == existing.NodeVersion &&
		s.Secured == existing.Secured &&
		s.Framework == existing.Framework &&
		s.PublicDir == existing.PublicDir &&
		s.Runtime == existing.Runtime &&
		s.RuntimeWorker == existing.RuntimeWorker &&
		s.ContainerPort == existing.ContainerPort &&
		s.ContainerSSL == existing.ContainerSSL &&
		s.HostPort == existing.HostPort &&
		s.HostSSL == existing.HostSSL &&
		s.HostCommand == existing.HostCommand
}

// sameDomainSet compares two domain lists as sets: a site serves the same
// domains whatever order they were resolved in.
func sameDomainSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for _, d := range a {
		if !slices.Contains(b, d) {
			return false
		}
	}
	return true
}
