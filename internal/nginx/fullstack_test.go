package nginx

import (
	"strings"
	"testing"
)

func TestRenderFullstack_PortBaseSiteRoutes(t *testing.T) {
	spec := ProxyVhostSpec{
		Domain:  "retencao.localhost",
		Secured: true,
		Base:    ProxyTarget{UpstreamHost: "host.containers.internal", UpstreamPort: 9000},
		Routes: []ProxyRouteSpec{
			{Path: "/api", Target: siteTarget("retencao-api", "/home/u/retencao-api/public", "82")},
			{Path: "/sanctum", Target: siteTarget("retencao-api", "/home/u/retencao-api/public", "82")},
		},
	}
	out, err := renderFullstackForTest(spec, "vhost-proxy-fullstack-ssl.conf.tmpl")
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	mustContain := []string{
		"server_name retencao.localhost;",
		"return 302 https://$host$request_uri;",
		"ssl_certificate /etc/nginx/certs/retencao.localhost.crt;",
		"location ^~ /api {",
		"location ^~ /sanctum {",
		"try_files $uri @site_retencao_api;",
		"location @site_retencao_api {",
		"fastcgi_pass $fpm:9000;",
		"lerd-php82-fpm",
		"fastcgi_param HTTP_HOST $real_forwarded_host;",
		"proxy_pass http://host.containers.internal:9000;",
	}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("output não contém %q\n---\n%s", s, out)
		}
	}
	// O bloco fastcgi do site deve aparecer UMA vez (dedup por named location).
	if n := strings.Count(out, "location @site_retencao_api {"); n != 1 {
		t.Errorf("named location aparece %d vezes, want 1", n)
	}
}

func siteTarget(name, docroot, phpShort string) ProxyTarget {
	return ProxyTarget{
		IsSite: true, SiteName: name, DocRoot: docroot,
		PHPShort: phpShort, LocationName: "site_" + strings.ReplaceAll(name, "-", "_"),
	}
}
