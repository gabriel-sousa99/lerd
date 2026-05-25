package proxyops

import (
	"fmt"

	"github.com/gabriel-sousa99/lerd/internal/certs"
	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/nginx"
)

// SetSecured flips the proxy's TLS state: issues/removes the certificate,
// regenerates the vhost, persists the registry, and reloads nginx. Single
// source of truth shared by CLI and UI.
func SetSecured(p *config.Proxy, secured bool) error {
	if secured {
		if err := secureCertFn(*p); err != nil {
			return fmt.Errorf("emitindo certificado: %w", err)
		}
	} else {
		if err := unsecureCertFn(*p); err != nil {
			return fmt.Errorf("removendo certificado: %w", err)
		}
	}
	p.Secured = secured
	if err := RegenerateProxyVhost(*p); err != nil {
		return fmt.Errorf("gerando vhost: %w", err)
	}
	if err := config.AddProxy(*p); err != nil {
		return fmt.Errorf("atualizando registry: %w", err)
	}
	_ = nginxReloadFn()
	return nil
}

// ApplyPause persists the Paused flag and ensures the vhost reflects it:
// when paused, the vhost is removed; when resumed, it is regenerated.
func ApplyPause(p *config.Proxy) error {
	if err := config.AddProxy(*p); err != nil {
		return err
	}
	if p.Paused {
		_ = nginx.RemoveVhost(p.PrimaryDomain())
	} else {
		if err := RegenerateProxyVhost(*p); err != nil {
			return err
		}
	}
	_ = nginxReloadFn()
	return nil
}

// StubForTests replaces external dependencies with no-ops so callers can
// exercise CLI/HTTP layers without invoking mkcert or systemd.
func StubForTests() {
	secureCertFn = func(config.Proxy) error { return nil }
	unsecureCertFn = func(config.Proxy) error { return nil }
	nginxReloadFn = func() error { return nil }
}

// UnstubForTests restores the real implementations.
func UnstubForTests() {
	secureCertFn = certs.SecureProxy
	unsecureCertFn = certs.UnsecureProxy
	nginxReloadFn = nginx.Reload
}
