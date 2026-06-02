package config

import (
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Route is one path-prefixed upstream of a fullstack proxy. Exactly one of
// Site (fastcgi to a lerd PHP site) or UpstreamPort (proxy_pass to host:port)
// must be set.
type Route struct {
	Path         string `yaml:"path"`
	Site         string `yaml:"site,omitempty"`
	UpstreamPort int    `yaml:"upstream_port,omitempty"`
	UpstreamHost string `yaml:"upstream_host,omitempty"`
}

// Proxy represents a manual reverse-proxy entry: a domain that nginx
// forwards to an arbitrary upstream (typically a SPA dev server on the
// host). Independent of `Site` (PHP) — different lifecycle, no framework,
// no worktrees, no env file.
type Proxy struct {
	Name         string   `yaml:"-"`
	Domains      []string `yaml:"-"`
	UpstreamPort int      `yaml:"upstream_port"`
	UpstreamHost string   `yaml:"upstream_host,omitempty"`
	Path         string   `yaml:"path,omitempty"`
	Secured      bool     `yaml:"secured"`
	Paused       bool     `yaml:"paused,omitempty"`

	Managed     bool   `yaml:"managed,omitempty"`
	NodeVersion string `yaml:"node_version,omitempty"`
	Command     string `yaml:"cmd,omitempty"`
	AutoStart   bool   `yaml:"auto_start,omitempty"`

	// Fullstack: Site routes the base (/) to a lerd PHP site instead of a
	// port; Routes maps extra path prefixes to their own upstreams. Empty
	// Routes == a plain single-upstream proxy (unchanged behaviour).
	Site   string  `yaml:"-"`
	Routes []Route `yaml:"-"`
}

// PrimaryDomain returns the first registered domain.
func (p Proxy) PrimaryDomain() string {
	if len(p.Domains) > 0 {
		return p.Domains[0]
	}
	return ""
}

// IsFullstack reports whether this proxy uses path-based routing.
func (p Proxy) IsFullstack() bool { return len(p.Routes) > 0 }

// ValidateProxyRoutes checks route paths and targets. Each path must start
// with "/", differ from "/", be unique, and carry exactly one target
// (Site xor UpstreamPort).
func ValidateProxyRoutes(routes []Route) error {
	seen := make(map[string]bool, len(routes))
	for _, r := range routes {
		if len(r.Path) == 0 || r.Path[0] != '/' {
			return fmt.Errorf("path da rota deve começar com /: %q", r.Path)
		}
		if r.Path == "/" {
			return fmt.Errorf("path da rota não pode ser / (reservado para a base)")
		}
		if seen[r.Path] {
			return fmt.Errorf("path de rota duplicado: %q", r.Path)
		}
		seen[r.Path] = true

		hasSite := r.Site != ""
		hasPort := r.UpstreamPort != 0
		if hasSite == hasPort {
			return fmt.Errorf("rota %q precisa de exatamente um target (site OU upstream_port)", r.Path)
		}
		if hasPort && (r.UpstreamPort <= 0 || r.UpstreamPort > 65535) {
			return fmt.Errorf("rota %q: porta inválida %d", r.Path, r.UpstreamPort)
		}
	}
	return nil
}

type proxyYAML struct {
	Name         string   `yaml:"name"`
	Domains      []string `yaml:"domains"`
	UpstreamPort int      `yaml:"upstream_port"`
	UpstreamHost string   `yaml:"upstream_host,omitempty"`
	Path         string   `yaml:"path,omitempty"`
	Secured      bool     `yaml:"secured"`
	Paused       bool     `yaml:"paused,omitempty"`
	Managed      bool     `yaml:"managed,omitempty"`
	NodeVersion  string   `yaml:"node_version,omitempty"`
	Command      string   `yaml:"cmd,omitempty"`
	AutoStart    bool     `yaml:"auto_start,omitempty"`
	Site         string   `yaml:"site,omitempty"`
	Routes       []Route  `yaml:"routes,omitempty"`
}

func (p Proxy) toYAML() proxyYAML {
	return proxyYAML{
		Name:         p.Name,
		Domains:      p.Domains,
		UpstreamPort: p.UpstreamPort,
		UpstreamHost: p.UpstreamHost,
		Path:         p.Path,
		Secured:      p.Secured,
		Paused:       p.Paused,
		Managed:      p.Managed,
		NodeVersion:  p.NodeVersion,
		Command:      p.Command,
		AutoStart:    p.AutoStart,
		Site:         p.Site,
		Routes:       p.Routes,
	}
}

func (py proxyYAML) toProxy() Proxy {
	return Proxy{
		Name:         py.Name,
		Domains:      py.Domains,
		UpstreamPort: py.UpstreamPort,
		UpstreamHost: py.UpstreamHost,
		Path:         py.Path,
		Secured:      py.Secured,
		Paused:       py.Paused,
		Managed:      py.Managed,
		NodeVersion:  py.NodeVersion,
		Command:      py.Command,
		AutoStart:    py.AutoStart,
		Site:         py.Site,
		Routes:       py.Routes,
	}
}

type ProxyRegistry struct{ Proxies []Proxy }

type proxyRegistryYAML struct {
	Proxies []proxyYAML `yaml:"proxies"`
}

var (
	proxiesCacheMu sync.Mutex
	proxiesCache   *ProxyRegistry
	proxiesCacheAt time.Time
	proxiesCacheSz int64
)

func invalidateProxiesCache() {
	proxiesCacheMu.Lock()
	proxiesCache = nil
	proxiesCacheAt = time.Time{}
	proxiesCacheSz = 0
	proxiesCacheMu.Unlock()
}

func cloneProxyRegistry(in *ProxyRegistry) *ProxyRegistry {
	if in == nil {
		return &ProxyRegistry{}
	}
	out := &ProxyRegistry{Proxies: make([]Proxy, len(in.Proxies))}
	for i, p := range in.Proxies {
		cp := p
		if p.Domains != nil {
			cp.Domains = append([]string(nil), p.Domains...)
		}
		if p.Routes != nil {
			cp.Routes = append([]Route(nil), p.Routes...)
		}
		out.Proxies[i] = cp
	}
	return out
}

func LoadProxies() (*ProxyRegistry, error) {
	path := ProxiesFile()
	info, statErr := os.Stat(path)

	proxiesCacheMu.Lock()
	if proxiesCache != nil && statErr == nil &&
		proxiesCacheAt.Equal(info.ModTime()) && proxiesCacheSz == info.Size() {
		out := cloneProxyRegistry(proxiesCache)
		proxiesCacheMu.Unlock()
		return out, nil
	}
	proxiesCacheMu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ProxyRegistry{}, nil
		}
		return nil, err
	}

	var raw proxyRegistryYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing proxies.yaml: %w", err)
	}
	reg := &ProxyRegistry{Proxies: make([]Proxy, 0, len(raw.Proxies))}
	for _, py := range raw.Proxies {
		reg.Proxies = append(reg.Proxies, py.toProxy())
	}

	proxiesCacheMu.Lock()
	proxiesCache = cloneProxyRegistry(reg)
	if statErr == nil {
		proxiesCacheAt = info.ModTime()
		proxiesCacheSz = info.Size()
	}
	proxiesCacheMu.Unlock()

	return reg, nil
}

func SaveProxies(reg *ProxyRegistry) error {
	raw := proxyRegistryYAML{Proxies: make([]proxyYAML, 0, len(reg.Proxies))}
	for _, p := range reg.Proxies {
		raw.Proxies = append(raw.Proxies, p.toYAML())
	}
	out, err := yaml.Marshal(raw)
	if err != nil {
		return err
	}
	path := ProxiesFile()
	if err := os.MkdirAll(DataDir(), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	invalidateProxiesCache()
	return nil
}

func AddProxy(p Proxy) error {
	reg, err := LoadProxies()
	if err != nil {
		return err
	}
	for i, existing := range reg.Proxies {
		if existing.Name == p.Name {
			reg.Proxies[i] = p
			return SaveProxies(reg)
		}
	}
	reg.Proxies = append(reg.Proxies, p)
	return SaveProxies(reg)
}

func RemoveProxy(name string) error {
	reg, err := LoadProxies()
	if err != nil {
		return err
	}
	out := reg.Proxies[:0]
	for _, p := range reg.Proxies {
		if p.Name != name {
			out = append(out, p)
		}
	}
	reg.Proxies = out
	return SaveProxies(reg)
}

func FindProxy(name string) (*Proxy, error) {
	reg, err := LoadProxies()
	if err != nil {
		return nil, err
	}
	for i := range reg.Proxies {
		if reg.Proxies[i].Name == name {
			return &reg.Proxies[i], nil
		}
	}
	return nil, fmt.Errorf("proxy %q not found", name)
}

func FindProxyByDomain(domain string) (*Proxy, error) {
	reg, err := LoadProxies()
	if err != nil {
		return nil, err
	}
	for i := range reg.Proxies {
		for _, d := range reg.Proxies[i].Domains {
			if d == domain {
				return &reg.Proxies[i], nil
			}
		}
	}
	return nil, fmt.Errorf("proxy with domain %q not found", domain)
}
