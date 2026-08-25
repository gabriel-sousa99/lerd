package proxyops

import (
	"fmt"
	"os"
	"strings"

	"github.com/gabriel-sousa99/lerd/internal/certs"
	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/nginx"
)

// AddOptions carries every parameter Add needs. Zero values are valid:
// Domain is required; Port is required; Path is optional unless Managed
// is true; NoSecure flips to plain HTTP; Managed/Command/Node/AutoStart
// are for the managed dev-server toggle.
type AddOptions struct {
	Domain         string
	Aliases        []string
	Port           int
	UpstreamHost   string
	UpstreamScheme string
	HealthPath     string
	TimeoutSeconds int
	Path           string
	NoSecure       bool
	Managed        bool
	Command        string
	NodeVersion    string
	AutoStart      bool

	// Fullstack: Site routes the base (/) to a lerd site; Routes maps path
	// prefixes to their own targets. Empty Routes+Site == simple proxy.
	Site   string
	Routes []config.Route
}

// Test hooks — production wires to the real packages.
var (
	secureCertFn  = certs.SecureProxy
	nginxReloadFn = nginx.Reload
)

// Add creates a new proxy entry, writes the vhost (HTTPS by default), and
// optionally issues a mkcert certificate. It does NOT start managed
// quadlets — that is the caller's responsibility (StartManaged).
func Add(opts AddOptions) (config.Proxy, error) {
	if opts.Domain == "" {
		return config.Proxy{}, fmt.Errorf("domínio é obrigatório")
	}
	// Porta da base é obrigatória só quando a base NÃO é um site.
	if opts.Site == "" && (opts.Port <= 0 || opts.Port > 65535) {
		return config.Proxy{}, fmt.Errorf("porta inválida: %d", opts.Port)
	}
	if opts.Managed && opts.Path == "" {
		return config.Proxy{}, fmt.Errorf("path obrigatório quando managed=true")
	}
	if opts.Path != "" {
		if _, err := os.Stat(opts.Path); err != nil {
			return config.Proxy{}, fmt.Errorf("path inválido: %w", err)
		}
	}

	domain := strings.ToLower(strings.TrimSpace(opts.Domain))
	domains := []string{domain}
	seen := map[string]bool{domain: true}
	for _, alias := range opts.Aliases {
		alias = strings.ToLower(strings.TrimSpace(alias))
		if alias != "" && !seen[alias] {
			seen[alias] = true
			domains = append(domains, alias)
		}
	}

	for _, candidate := range domains {
		if existing, err := config.FindProxyByDomain(candidate); err == nil && existing != nil {
			return config.Proxy{}, fmt.Errorf("já existe um proxy para %s", candidate)
		}
		if site, err := config.FindSiteByDomain(candidate); err == nil && site != nil {
			return config.Proxy{}, fmt.Errorf("domínio %s já está registrado como site PHP (%s)", candidate, site.Name)
		}
	}

	name, _ := ProxyNameAndDomain(domain, "")

	p := config.Proxy{
		Name:           name,
		Domains:        domains,
		UpstreamPort:   opts.Port,
		UpstreamHost:   opts.UpstreamHost,
		UpstreamScheme: opts.UpstreamScheme,
		HealthPath:     opts.HealthPath,
		TimeoutSeconds: opts.TimeoutSeconds,
		Path:           opts.Path,
		Secured:        !opts.NoSecure,
		Managed:        opts.Managed,
		Command:        opts.Command,
		NodeVersion:    opts.NodeVersion,
		AutoStart:      opts.AutoStart,
		Site:           opts.Site,
		Routes:         opts.Routes,
	}

	if err := p.Validate(); err != nil {
		return config.Proxy{}, err
	}

	if p.Secured {
		if err := secureCertFn(p); err != nil {
			return config.Proxy{}, fmt.Errorf("emitindo certificado: %w", err)
		}
	}
	if err := RegenerateProxyVhost(p); err != nil {
		return config.Proxy{}, fmt.Errorf("gerando vhost: %w", err)
	}
	if err := config.AddProxy(p); err != nil {
		return config.Proxy{}, fmt.Errorf("salvando registry: %w", err)
	}
	// Reload é best-effort: em testes / antes do nginx subir é normal falhar.
	_ = nginxReloadFn()
	if p.IsFullstack() {
		_ = syncProxyEnv(p)
	}
	return p, nil
}
