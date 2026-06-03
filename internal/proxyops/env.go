package proxyops

import (
	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/envfile"
)

// syncEnvFn is injectable for tests. Production wires to envfile.SyncPrimaryDomain.
var syncEnvFn = envfile.SyncPrimaryDomain

// Frontend env-sync hooks, injectable for tests.
var (
	syncFrontendFn   = envfile.SyncFrontendAPIBase
	revertFrontendFn = envfile.RevertFrontendAPIBase
)

// boundSites returns the distinct site names a proxy targets (base + routes).
func boundSites(p config.Proxy) []string {
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	add(p.Site)
	for _, r := range p.Routes {
		add(r.Site)
	}
	return out
}

// syncProxyEnv points the .env of every site bound to p at the unified domain
// (p's primary domain), so sessions/cookies are first-party. When p.Path is
// set (the frontend project folder), its API-base keys are pointed at the same
// unified origin so the SPA stops calling a cross-origin API. Best-effort: a
// missing site or .env is skipped without failing the proxy operation.
func syncProxyEnv(p config.Proxy) error {
	domain := p.PrimaryDomain()
	for _, name := range boundSites(p) {
		s, err := findSiteFn(name)
		if err != nil {
			continue
		}
		_ = syncEnvFn(s.Path, domain, p.Secured)
	}
	if p.Path != "" {
		_ = syncFrontendFn(p.Path, domain, p.Secured)
	}
	return nil
}

// unbindSitesEnv reverts each named site's .env back to its own primary domain.
func unbindSitesEnv(names []string) {
	for _, name := range names {
		s, err := findSiteFn(name)
		if err != nil {
			continue
		}
		_ = syncEnvFn(s.Path, s.PrimaryDomain(), s.Secured)
	}
}
