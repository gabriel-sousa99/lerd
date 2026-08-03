package nginx

import (
	"strings"
	"testing"
)

// nginx splits a directive on whitespace, so an unquoted root under a path with
// a space gives root three arguments and nginx rejects the whole config, taking
// every other site down with it. The fullstack vhost has to quote its doc roots
// the way VhostData.Root does for site vhosts.
func TestFullstackVhost_quotesDocRootWithSpace(t *testing.T) {
	spec := ProxyVhostSpec{
		Domain: "app.localhost",
		Base: ProxyTarget{
			IsSite: true, DocRoot: "/home/u/Meus Projetos/api/public",
			PHPShort: "84", SiteName: "api", LocationName: "site_api",
		},
		Routes: []ProxyRouteSpec{{
			Path: "/api",
			Target: ProxyTarget{
				IsSite: true, DocRoot: "/home/u/Meus Projetos/api/public",
				PHPShort: "84", SiteName: "api", LocationName: "site_api",
			},
		}},
	}
	for _, tmpl := range []string{"vhost-proxy-fullstack.conf.tmpl", "vhost-proxy-fullstack-ssl.conf.tmpl"} {
		out, err := renderFullstackForTest(spec, tmpl)
		if err != nil {
			t.Fatalf("%s: render: %v", tmpl, err)
		}
		for _, line := range strings.Split(out, "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "root ") {
				continue
			}
			// A quoted token is one argument; an unquoted path with a space is three.
			if !strings.Contains(trimmed, `root "`) {
				t.Errorf("%s: unquoted root with a space would break every vhost: %q", tmpl, trimmed)
			}
		}
	}
}

// A value that could close the directive or the server block must be refused
// before it is written, matching VhostData.validate for site vhosts.
func TestFullstackVhost_refusesDirectiveBreakingValues(t *testing.T) {
	base := func() ProxyVhostSpec {
		return ProxyVhostSpec{
			Domain: "app.localhost",
			Base: ProxyTarget{
				IsSite: true, DocRoot: "/home/u/api/public",
				PHPShort: "84", SiteName: "api", LocationName: "site_api",
			},
		}
	}
	cases := map[string]func(*ProxyVhostSpec){
		"domain closes the server block":  func(s *ProxyVhostSpec) { s.Domain = "app.localhost; } server { listen 80" },
		"domain comments out the rest":    func(s *ProxyVhostSpec) { s.Domain = "app.localhost #" },
		"site name breaks fastcgi_param":  func(s *ProxyVhostSpec) { s.Base.SiteName = "api; return 200 'pwned'" },
		"upstream host breaks proxy_pass": func(s *ProxyVhostSpec) { s.Base = ProxyTarget{UpstreamHost: "127.0.0.1; }", UpstreamPort: 3000} },
		"route path breaks location": func(s *ProxyVhostSpec) {
			s.Routes = []ProxyRouteSpec{{Path: "/api { deny all; } location /x", Target: ProxyTarget{UpstreamPort: 3000, UpstreamHost: "127.0.0.1"}}}
		},
		"doc root closes the server block": func(s *ProxyVhostSpec) { s.Base.DocRoot = "/home/u/api/public\"; }" },
	}
	for name, mutate := range cases {
		s := base()
		mutate(&s)
		if err := s.validate(); err == nil {
			t.Errorf("%s: validate() = nil; want rejection", name)
		}
	}
}

// The legitimate shapes must keep validating, including a path with a space.
func TestFullstackVhost_validateAcceptsLegitimate(t *testing.T) {
	for _, s := range []ProxyVhostSpec{
		{Domain: "app.localhost", Base: ProxyTarget{UpstreamHost: "host.containers.internal", UpstreamPort: 5173}},
		{Domain: "app.localhost", Secured: true,
			Base: ProxyTarget{IsSite: true, DocRoot: "/home/u/Meus Projetos/api/public",
				PHPShort: "84", SiteName: "api", LocationName: "site_api"},
			Routes: []ProxyRouteSpec{{Path: "/api", Target: ProxyTarget{UpstreamHost: "host.containers.internal", UpstreamPort: 8000}}}},
	} {
		if err := s.validate(); err != nil {
			t.Errorf("validate() = %v for legitimate spec %q; want nil", err, s.Domain)
		}
	}
}
