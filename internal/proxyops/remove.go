package proxyops

import (
	"github.com/gabriel-sousa99/lerd/internal/certs"
	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/nginx"
)

// Test hook.
var unsecureCertFn = certs.UnsecureProxy

// Remove deletes the proxy, removes its vhost + cert files.
func Remove(name string) error {
	p, err := config.FindProxy(name)
	if err != nil {
		return err
	}
	_ = nginx.RemoveVhost(p.PrimaryDomain())
	if p.Secured {
		_ = unsecureCertFn(*p)
	}
	if err := config.RemoveProxy(name); err != nil {
		return err
	}
	_ = nginxReloadFn()
	return nil
}
