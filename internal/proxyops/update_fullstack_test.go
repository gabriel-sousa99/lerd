package proxyops

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestUpdate_ReplacesRoutesAndUnbindsRemoved(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	origFind, origSync, origReload := findSiteFn, syncEnvFn, nginxReloadFn
	defer func() { findSiteFn, syncEnvFn, nginxReloadFn = origFind, origSync, origReload }()
	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name, Domains: []string{name + ".localhost"}, PublicDir: "public", PHPVersion: "8.2", Secured: true}, nil
	}
	var synced []string
	syncEnvFn = func(path, domain string, secured bool) error { synced = append(synced, domain); return nil }
	nginxReloadFn = func() error { return nil }

	if err := config.AddProxy(config.Proxy{
		Name: "app", Domains: []string{"app.localhost"}, UpstreamPort: 9000, Secured: true,
		Routes: []config.Route{{Path: "/api", Site: "old-api"}},
	}); err != nil {
		t.Fatal(err)
	}

	newRoutes := []config.Route{{Path: "/api", Site: "new-api"}}
	p, err := Update("app", UpdateOptions{Routes: &newRoutes})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(p.Routes) != 1 || p.Routes[0].Site != "new-api" {
		t.Errorf("routes = %+v", p.Routes)
	}
	var sawUnified, sawRevert bool
	for _, d := range synced {
		if d == "app.localhost" {
			sawUnified = true
		}
		if d == "old-api.localhost" {
			sawRevert = true
		}
	}
	if !sawUnified || !sawRevert {
		t.Errorf("synced = %v (want unified app.localhost + revert old-api.localhost)", synced)
	}
}
