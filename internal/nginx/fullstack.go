package nginx

import (
	"bytes"
	"os"
	"path/filepath"
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

// ProxyRouteSpec binds a path prefix to a resolved target.
type ProxyRouteSpec struct {
	Path   string
	Target ProxyTarget
}

// ProxyVhostSpec is the fully-resolved input for a fullstack proxy vhost.
type ProxyVhostSpec struct {
	Domain  string
	Secured bool
	Base    ProxyTarget      // catch-all "/"
	Routes  []ProxyRouteSpec // path-prefixed routes
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
	if err := os.WriteFile(filepath.Join(config.NginxConfD(), confName), buf.Bytes(), 0644); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(config.NginxConfD(), stale))
	return nil
}

// renderFullstackForTest renders a named fullstack template to a string
// without touching the filesystem (used by tests).
func renderFullstackForTest(spec ProxyVhostSpec, name string) (string, error) {
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
