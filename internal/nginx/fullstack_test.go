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

func TestRenderFullstack_PortRouteHasConnectTimeout(t *testing.T) {
	spec := ProxyVhostSpec{
		Domain:  "app.localhost",
		Secured: true,
		Base:    ProxyTarget{UpstreamHost: "host.containers.internal", UpstreamPort: 9000},
		Routes: []ProxyRouteSpec{
			{Path: "/legacy", Target: ProxyTarget{UpstreamHost: "127.0.0.1", UpstreamPort: 8001}},
		},
	}
	out, err := renderFullstackForTest(spec, "vhost-proxy-fullstack-ssl.conf.tmpl")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out, "location ^~ /legacy {") {
		t.Errorf("faltou location da rota porta\n%s", out)
	}
	if !strings.Contains(out, "proxy_pass http://127.0.0.1:8001;") {
		t.Errorf("faltou proxy_pass da rota porta\n%s", out)
	}
	// A rota com target porta deve ter o mesmo connect timeout da base.
	if n := strings.Count(out, "proxy_connect_timeout 5s;"); n < 2 {
		t.Errorf("proxy_connect_timeout aparece %d vezes, want >= 2 (base + rota)", n)
	}
}

func TestRenderFullstack_SiteBase(t *testing.T) {
	spec := ProxyVhostSpec{
		Domain:  "blog.localhost",
		Secured: true,
		Base:    siteTarget("blog", "/home/u/blog/public", "84"),
	}
	out, err := renderFullstackForTest(spec, "vhost-proxy-fullstack-ssl.conf.tmpl")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{
		"root /home/u/blog/public;", // server-level root for site base
		"index index.php index.html;",
		"try_files $uri $uri/ /index.php?$query_string;", // catch-all for site base
		"location ~ \\.php$ {",                           // base php handler
		"lerd-php84-fpm",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output não contém %q\n---\n%s", want, out)
		}
	}
}

func siteTarget(name, docroot, phpShort string) ProxyTarget {
	return ProxyTarget{
		IsSite: true, SiteName: name, DocRoot: docroot,
		PHPShort: phpShort, LocationName: "site_" + strings.ReplaceAll(name, "-", "_"),
	}
}
