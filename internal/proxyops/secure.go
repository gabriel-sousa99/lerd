package proxyops

import (
	"fmt"

	"github.com/gabriel-sousa99/lerd/internal/config"
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
	if err := config.AddProxy(*p); err != nil {
		return fmt.Errorf("atualizando registry: %w", err)
	}
	if err := RegenerateProxyVhost(*p); err != nil {
		return fmt.Errorf("gerando vhost: %w", err)
	}
	_ = nginxReloadFn()
	return nil
}
