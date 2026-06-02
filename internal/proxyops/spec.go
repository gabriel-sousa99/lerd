package proxyops

import (
	"strings"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/nginx"
)

// findSiteFn is injectable so tests can resolve sites without a real
// sites.yaml. Production wires to config.FindSite.
var findSiteFn = config.FindSite

// resolveProxySpec turns a fullstack config.Proxy into the fully-resolved
// data the nginx package needs: every site target carries an absolute
// docroot and PHP short version; routes that hit the same site share one
// named location so the fastcgi block is emitted once.
func resolveProxySpec(p config.Proxy) (nginx.ProxyVhostSpec, error) {
	if err := config.ValidateProxyRoutes(p.Routes); err != nil {
		return nginx.ProxyVhostSpec{}, err
	}

	base, err := resolveTarget(p.Site, p.UpstreamPort, p.UpstreamHost)
	if err != nil {
		return nginx.ProxyVhostSpec{}, err
	}

	spec := nginx.ProxyVhostSpec{
		Domain:  p.PrimaryDomain(),
		Secured: p.Secured,
		Base:    base,
	}
	for _, r := range p.Routes {
		t, err := resolveTarget(r.Site, r.UpstreamPort, r.UpstreamHost)
		if err != nil {
			return nginx.ProxyVhostSpec{}, err
		}
		spec.Routes = append(spec.Routes, nginx.ProxyRouteSpec{Path: r.Path, Target: t})
	}
	return spec, nil
}

// resolveTarget builds a nginx.ProxyTarget from a (site | port) pair. site
// wins when set; otherwise it's a port target defaulting the host.
func resolveTarget(site string, port int, host string) (nginx.ProxyTarget, error) {
	if site != "" {
		s, err := findSiteFn(site)
		if err != nil {
			return nginx.ProxyTarget{}, err
		}
		public := s.PublicDir
		if public == "" {
			public = "public"
		}
		return nginx.ProxyTarget{
			IsSite:       true,
			SiteName:     s.Name,
			DocRoot:      strings.TrimRight(s.Path, "/") + "/" + public,
			PHPShort:     strings.ReplaceAll(s.PHPVersion, ".", ""),
			LocationName: sanitizeLocation(s.Name),
		}, nil
	}
	h := host
	if h == "" {
		h = defaultUpstreamHost
	}
	return nginx.ProxyTarget{UpstreamPort: port, UpstreamHost: h}, nil
}

// sanitizeLocation maps a site name to a safe nginx named-location label,
// e.g. "retencao-api" -> "site_retencao_api".
func sanitizeLocation(name string) string {
	var b strings.Builder
	b.WriteString("site_")
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
