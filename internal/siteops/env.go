package siteops

import (
	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/envfile"
)

// Injectable seams for tests.
var (
	findFullstackFn = config.FindFullstackProxyForSite
	syncEnvFn       = envfile.SyncPrimaryDomain
)

// effectiveEnvDomain returns the domain/TLS that the site's .env should
// reflect: the unified domain of the fullstack proxy it's bound to (if any),
// otherwise its own primary domain.
func effectiveEnvDomain(site config.Site) (string, bool) {
	if p, ok := findFullstackFn(site.Name); ok {
		return p.PrimaryDomain(), p.Secured
	}
	return site.PrimaryDomain(), site.Secured
}

// SyncSiteEnv syncs the site's domain-scoped .env keys to the effective
// domain (fullstack-aware). Drop-in replacement for direct calls to
// envfile.SyncPrimaryDomain(site.Path, site.PrimaryDomain(), site.Secured).
func SyncSiteEnv(site config.Site) error {
	domain, secured := effectiveEnvDomain(site)
	return syncEnvFn(site.Path, domain, secured)
}
