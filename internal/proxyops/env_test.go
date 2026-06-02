package proxyops

import (
	"sort"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestBoundSites(t *testing.T) {
	p := config.Proxy{
		Site: "spa", UpstreamPort: 0,
		Routes: []config.Route{
			{Path: "/api", Site: "api"},
			{Path: "/sanctum", Site: "api"},
			{Path: "/legacy", UpstreamPort: 8001},
		},
	}
	got := boundSites(p)
	sort.Strings(got)
	want := []string{"api", "spa"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("boundSites = %v, want %v", got, want)
	}
}

func TestSyncProxyEnv_PointsSitesToUnifiedDomain(t *testing.T) {
	origFind := findSiteFn
	origSync := syncEnvFn
	defer func() { findSiteFn = origFind; syncEnvFn = origSync }()

	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name}, nil
	}
	type call struct {
		path, domain string
		secured      bool
	}
	var calls []call
	syncEnvFn = func(path, domain string, secured bool) error {
		calls = append(calls, call{path, domain, secured})
		return nil
	}

	p := config.Proxy{
		Domains: []string{"retencao.localhost"}, UpstreamPort: 9000, Secured: true,
		Routes: []config.Route{{Path: "/api", Site: "retencao-api"}},
	}
	if err := syncProxyEnv(p); err != nil {
		t.Fatalf("syncProxyEnv: %v", err)
	}
	if len(calls) != 1 || calls[0].path != "/srv/retencao-api" ||
		calls[0].domain != "retencao.localhost" || !calls[0].secured {
		t.Fatalf("calls = %+v", calls)
	}
}

func TestUnbindSitesEnv_RevertsToOwnDomain(t *testing.T) {
	origFind := findSiteFn
	origSync := syncEnvFn
	defer func() { findSiteFn = origFind; syncEnvFn = origSync }()

	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name, Domains: []string{name + ".localhost"}, Secured: true}, nil
	}
	var gotDomain string
	syncEnvFn = func(path, domain string, secured bool) error { gotDomain = domain; return nil }

	unbindSitesEnv([]string{"retencao-api"})
	if gotDomain != "retencao-api.localhost" {
		t.Errorf("domain = %q, want retencao-api.localhost", gotDomain)
	}
}
