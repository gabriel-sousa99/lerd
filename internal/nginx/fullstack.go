package nginx

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// ProxyTarget is a resolved upstream: either a lerd PHP site (fastcgi) or a
// host:port (proxy_pass). proxyops fills this from config.
type ProxyTarget struct {
	IsSite       bool
	UpstreamHost string // port mode
	UpstreamPort int    // port mode
	DocRoot      string // site mode: absolute docroot (path/publicdir)
	PHPShort     string // site mode: e.g. "82"
	SiteName     string // site mode
	LocationName string // site mode: named location label (e.g. site_retencao_api)
}

// Root is the document root as the templates render it, quoted so a path with a
// space stays a single nginx token. Mirrors VhostData.Root, which every site
// vhost goes through.
func (t ProxyTarget) Root() string {
	return nginxQuote(t.DocRoot)
}

// ProxyRouteSpec binds a path prefix to a resolved target.
type ProxyRouteSpec struct {
	Path   string
	Target ProxyTarget
}

// ProxyVhostSpec is the fully-resolved input for a fullstack proxy vhost.
type ProxyVhostSpec struct {
	Domain         string
	Domains        []string
	Secured        bool
	UpstreamScheme string
	RequestTimeout int
	Base           ProxyTarget      // catch-all "/"
	Routes         []ProxyRouteSpec // path-prefixed routes
}

func (s ProxyVhostSpec) withDefaults() ProxyVhostSpec {
	if len(s.Domains) == 0 {
		s.Domains = []string{s.Domain}
	}
	if s.UpstreamScheme == "" {
		s.UpstreamScheme = "http"
	}
	if s.RequestTimeout <= 0 {
		s.RequestTimeout = 86400
	}
	return s
}

// validate refuses any value that could break out of the directive it lands in,
// the same rule VhostData.validate applies to every site vhost. The doc roots
// are quoted rather than rejected for whitespace, so only the characters that
// end a directive or open a block are refused.
func (s ProxyVhostSpec) validate() error {
	s = s.withDefaults()
	values := map[string]string{
		"domain": s.Domain, "upstream scheme": s.UpstreamScheme,
	}
	for i, domain := range s.Domains {
		values[fmt.Sprintf("domain %d", i+1)] = domain
	}
	addTarget := func(label string, t ProxyTarget) {
		values[label+" doc root"] = t.DocRoot
		values[label+" upstream host"] = t.UpstreamHost
		values[label+" site name"] = t.SiteName
		values[label+" location name"] = t.LocationName
		values[label+" php version"] = t.PHPShort
	}
	addTarget("base", s.Base)
	for i, r := range s.Routes {
		label := fmt.Sprintf("route %d", i+1)
		values[label+" path"] = r.Path
		addTarget(label, r.Target)
	}

	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		v := values[name]
		if i := strings.IndexAny(v, nginxValueForbidden); i >= 0 {
			return fmt.Errorf("nginx %s %q contains %q, which would end the directive it lands in", name, v, string(v[i]))
		}
	}
	if s.UpstreamScheme != "http" && s.UpstreamScheme != "https" {
		return fmt.Errorf("invalid upstream scheme %q", s.UpstreamScheme)
	}
	if s.RequestTimeout <= 0 || s.RequestTimeout > 86400 {
		return fmt.Errorf("invalid proxy timeout %d", s.RequestTimeout)
	}
	// A quote in a doc root would close the quoted token Root emits.
	for name, v := range map[string]string{"base doc root": s.Base.DocRoot} {
		if strings.Contains(v, `"`) {
			return fmt.Errorf("nginx %s %q contains a quote, which would close the quoted token", name, v)
		}
	}
	for i, r := range s.Routes {
		if strings.Contains(r.Target.DocRoot, `"`) {
			return fmt.Errorf("nginx route %d doc root %q contains a quote, which would close the quoted token", i+1, r.Target.DocRoot)
		}
	}
	return nil
}

// siteLocations dedups the named locations needed for site targets so the
// fastcgi block for a given site is emitted exactly once.
func (s ProxyVhostSpec) siteLocations() []ProxyTarget {
	seen := map[string]bool{}
	var out []ProxyTarget
	consider := func(t ProxyTarget) {
		if t.IsSite && !seen[t.LocationName] {
			seen[t.LocationName] = true
			out = append(out, t)
		}
	}
	consider(s.Base)
	for _, r := range s.Routes {
		consider(r.Target)
	}
	return out
}

// GenerateFullstackProxyVhost renders the fullstack template for spec and
// writes conf.d/<domain>.conf (HTTP) or <domain>-ssl.conf (HTTPS),
// removing the stale counterpart like GenerateProxyVhost does.
func GenerateFullstackProxyVhost(spec ProxyVhostSpec) error {
	spec = spec.withDefaults()
	if err := spec.validate(); err != nil {
		return err
	}
	tmplName := "vhost-proxy-fullstack.conf.tmpl"
	confName := spec.Domain + ".conf"
	stale := spec.Domain + "-ssl.conf"
	if spec.Secured {
		tmplName = "vhost-proxy-fullstack-ssl.conf.tmpl"
		confName = spec.Domain + "-ssl.conf"
		stale = spec.Domain + ".conf"
	}

	tmplData, err := GetTemplate(tmplName)
	if err != nil {
		return err
	}
	tmpl, err := template.New(tmplName).Funcs(template.FuncMap{
		"siteLocations": spec.siteLocations,
	}).Parse(string(tmplData))
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, spec); err != nil {
		return err
	}
	if err := os.MkdirAll(config.NginxConfD(), 0755); err != nil {
		return err
	}
	confPath := filepath.Join(config.NginxConfD(), confName)
	config.GuardRealWrite(confPath)
	if err := os.WriteFile(confPath, buf.Bytes(), 0644); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(config.NginxConfD(), stale))
	return nil
}

// renderFullstackForTest renders a named fullstack template to a string
// without touching the filesystem (used by tests).
func renderFullstackForTest(spec ProxyVhostSpec, name string) (string, error) {
	spec = spec.withDefaults()
	tmplData, err := GetTemplate(name)
	if err != nil {
		return "", err
	}
	tmpl, err := template.New(name).Funcs(template.FuncMap{
		"siteLocations": spec.siteLocations,
	}).Parse(string(tmplData))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, spec); err != nil {
		return "", err
	}
	return buf.String(), nil
}
