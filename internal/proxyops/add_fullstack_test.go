package proxyops

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestAdd_Fullstack_SyncsEnvAndPersistsRoutes(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	origFind, origSync, origReload, origSecure := findSiteFn, syncEnvFn, nginxReloadFn, secureCertFn
	defer func() {
		findSiteFn, syncEnvFn, nginxReloadFn, secureCertFn = origFind, origSync, origReload, origSecure
	}()
	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name, PublicDir: "public", PHPVersion: "8.2"}, nil
	}
	var syncedDomain string
	syncEnvFn = func(path, domain string, secured bool) error { syncedDomain = domain; return nil }
	nginxReloadFn = func() error { return nil }
	secureCertFn = func(p config.Proxy) error { return nil }

	p, err := Add(AddOptions{
		Domain: "retencao.localhost", Port: 9000,
		Routes: []config.Route{{Path: "/api", Site: "retencao-api"}},
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !p.IsFullstack() || len(p.Routes) != 1 || p.Routes[0].Site != "retencao-api" {
		t.Errorf("proxy = %+v", p)
	}
	if syncedDomain != "retencao.localhost" {
		t.Errorf("env synced to %q, want retencao.localhost", syncedDomain)
	}
	got, err := config.FindProxy(p.Name)
	if err != nil || len(got.Routes) != 1 {
		t.Errorf("persisted = %+v err=%v", got, err)
	}
}
