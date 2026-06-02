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

func TestUpdate_ClearRoutesRevertsAllSites(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	origFind, origSync, origReload := findSiteFn, syncEnvFn, nginxReloadFn
	defer func() { findSiteFn, syncEnvFn, nginxReloadFn = origFind, origSync, origReload }()
	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name, Domains: []string{name + ".localhost"}, Secured: true}, nil
	}
	var reverted []string
	syncEnvFn = func(path, domain string, secured bool) error { reverted = append(reverted, domain); return nil }
	nginxReloadFn = func() error { return nil }

	if err := config.AddProxy(config.Proxy{
		Name: "app", Domains: []string{"app.localhost"}, UpstreamPort: 9000, Secured: true,
		Routes: []config.Route{{Path: "/api", Site: "api-a"}, {Path: "/admin", Site: "api-b"}},
	}); err != nil {
		t.Fatal(err)
	}

	empty := []config.Route{}
	p, err := Update("app", UpdateOptions{Routes: &empty})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if p.IsFullstack() {
		t.Errorf("após limpar rotas, proxy deveria ser simples: %+v", p.Routes)
	}
	// Ambos os sites antes vinculados voltam ao próprio domínio.
	var sawA, sawB bool
	for _, d := range reverted {
		if d == "api-a.localhost" {
			sawA = true
		}
		if d == "api-b.localhost" {
			sawB = true
		}
	}
	if !sawA || !sawB {
		t.Errorf("esperava reverter api-a e api-b; reverted=%v", reverted)
	}
}

func TestUpdate_InvalidRouteReplacementRejectedNotPersisted(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	origReload := nginxReloadFn
	defer func() { nginxReloadFn = origReload }()
	nginxReloadFn = func() error { return nil }

	if err := config.AddProxy(config.Proxy{
		Name: "app", Domains: []string{"app.localhost"}, UpstreamPort: 9000, Secured: true,
		Routes: []config.Route{{Path: "/api", Site: "api-a"}},
	}); err != nil {
		t.Fatal(err)
	}

	// Rota sem target (nem site nem porta) é inválida.
	bad := []config.Route{{Path: "/api"}}
	if _, err := Update("app", UpdateOptions{Routes: &bad}); err == nil {
		t.Fatal("esperava erro de validação para rota sem target")
	}
	// O registry NÃO deve ter sido alterado.
	got, err := config.FindProxy("app")
	if err != nil {
		t.Fatalf("FindProxy: %v", err)
	}
	if len(got.Routes) != 1 || got.Routes[0].Site != "api-a" {
		t.Errorf("rota original deveria persistir intacta; got=%+v", got.Routes)
	}
}
