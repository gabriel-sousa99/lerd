package proxyops

import (
	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/nginx"
)

const defaultUpstreamHost = "host.containers.internal"

// upstreamHost returns the host used in proxy_pass: the explicit value
// from the proxy if set, otherwise host.containers.internal (so the
// nginx container reaches dev servers running on the host).
func upstreamHost(p config.Proxy) string {
	if p.UpstreamHost != "" {
		return p.UpstreamHost
	}
	return defaultUpstreamHost
}

// RegenerateProxyVhost writes the nginx config for p based on Secured. For
// fullstack proxies (p.IsFullstack()) it resolves the route spec and renders
// the fullstack template; otherwise it keeps the simple single-upstream path
// (byte-identical to before).
func RegenerateProxyVhost(p config.Proxy) error {
	if p.IsFullstack() {
		spec, err := resolveProxySpec(p)
		if err != nil {
			return err
		}
		return nginx.GenerateFullstackProxyVhost(spec)
	}
	return nginx.GenerateProxyVhost(p.PrimaryDomain(), upstreamHost(p), p.UpstreamPort, p.Secured)
}
