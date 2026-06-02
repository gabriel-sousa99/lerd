package proxyops

import (
	"fmt"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestResolveProxySpec_PortBaseAndSiteRoute(t *testing.T) {
	orig := findSiteFn
	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{
			Name:       "retencao-api",
			Domains:    []string{"retencao-api.localhost"},
			Path:       "/home/user/retencao-api",
			PublicDir:  "public",
			PHPVersion: "8.2",
		}, nil
	}
	defer func() { findSiteFn = orig }()

	p := config.Proxy{
		Name:         "retencao",
		Domains:      []string{"retencao.localhost"},
		UpstreamPort: 9000,
		Secured:      true,
		Routes: []config.Route{
			{Path: "/api", Site: "retencao-api"},
			{Path: "/sanctum", Site: "retencao-api"},
			{Path: "/legacy", UpstreamPort: 8001},
		},
	}

	spec, err := resolveProxySpec(p)
	if err != nil {
		t.Fatalf("resolveProxySpec: %v", err)
	}
	if spec.Domain != "retencao.localhost" || !spec.Secured {
		t.Errorf("spec = %+v", spec)
	}
	if spec.Base.IsSite || spec.Base.UpstreamPort != 9000 || spec.Base.UpstreamHost != "host.containers.internal" {
		t.Errorf("base = %+v", spec.Base)
	}
	if len(spec.Routes) != 3 {
		t.Fatalf("routes = %d, want 3", len(spec.Routes))
	}
	if spec.Routes[0].Target.LocationName != spec.Routes[1].Target.LocationName {
		t.Errorf("rotas do mesmo site deveriam compartilhar location: %q vs %q",
			spec.Routes[0].Target.LocationName, spec.Routes[1].Target.LocationName)
	}
	r0 := spec.Routes[0].Target
	if !r0.IsSite || r0.DocRoot != "/home/user/retencao-api/public" || r0.PHPShort != "82" {
		t.Errorf("route[0].target = %+v", r0)
	}
	if r0.LocationName != "site_retencao_api" {
		t.Errorf("location name = %q, want site_retencao_api", r0.LocationName)
	}
	r2 := spec.Routes[2].Target
	if r2.IsSite || r2.UpstreamPort != 8001 || r2.UpstreamHost != "host.containers.internal" {
		t.Errorf("route[2].target = %+v", r2)
	}
}

func TestResolveProxySpec_SiteNotFound(t *testing.T) {
	orig := findSiteFn
	findSiteFn = func(name string) (*config.Site, error) {
		return nil, fmt.Errorf("not found")
	}
	defer func() { findSiteFn = orig }()

	p := config.Proxy{
		Name: "x", Domains: []string{"x.localhost"}, UpstreamPort: 9000,
		Routes: []config.Route{{Path: "/api", Site: "missing"}},
	}
	if _, err := resolveProxySpec(p); err == nil {
		t.Fatal("esperava erro para site inexistente")
	}
}

func TestResolveProxySpec_PublicDirDefault(t *testing.T) {
	orig := findSiteFn
	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: "api", Path: "/srv/api", PublicDir: "", PHPVersion: "8.3"}, nil
	}
	defer func() { findSiteFn = orig }()

	p := config.Proxy{
		Name: "app", Domains: []string{"app.localhost"}, UpstreamPort: 9000,
		Routes: []config.Route{{Path: "/api", Site: "api"}},
	}
	spec, err := resolveProxySpec(p)
	if err != nil {
		t.Fatalf("resolveProxySpec: %v", err)
	}
	if spec.Routes[0].Target.DocRoot != "/srv/api/public" {
		t.Errorf("docroot = %q, want /srv/api/public (PublicDir default)", spec.Routes[0].Target.DocRoot)
	}
}
