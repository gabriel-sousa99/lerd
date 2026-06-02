package proxyops

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestRemove_Fullstack_RevertsSiteEnv(t *testing.T) {
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
		Routes: []config.Route{{Path: "/api", Site: "api"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := Remove("app"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	found := false
	for _, d := range reverted {
		if d == "api.localhost" {
			found = true
		}
	}
	if !found {
		t.Errorf("env do site não revertido ao próprio domínio; calls=%v", reverted)
	}
}
