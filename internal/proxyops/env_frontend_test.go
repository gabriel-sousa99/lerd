package proxyops

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestSyncProxyEnv_SyncsFrontendWhenPathSet(t *testing.T) {
	origFind, origSync, origFront := findSiteFn, syncEnvFn, syncFrontendFn
	defer func() { findSiteFn, syncEnvFn, syncFrontendFn = origFind, origSync, origFront }()

	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name}, nil
	}
	syncEnvFn = func(path, domain string, secured bool) error { return nil }

	var frontPath, frontDomain string
	var frontSecured bool
	syncFrontendFn = func(path, domain string, secured bool) error {
		frontPath, frontDomain, frontSecured = path, domain, secured
		return nil
	}

	p := config.Proxy{
		Domains: []string{"gestao-clientes.localhost"}, UpstreamPort: 9000,
		Path: "/srv/spa", Secured: true,
		Routes: []config.Route{{Path: "/api", Site: "gc-api"}},
	}
	if err := syncProxyEnv(p); err != nil {
		t.Fatalf("syncProxyEnv: %v", err)
	}
	if frontPath != "/srv/spa" || frontDomain != "gestao-clientes.localhost" || !frontSecured {
		t.Errorf("frontend sync = (%q, %q, %v)", frontPath, frontDomain, frontSecured)
	}
}

func TestSyncProxyEnv_NoFrontendWhenPathEmpty(t *testing.T) {
	origFind, origSync, origFront := findSiteFn, syncEnvFn, syncFrontendFn
	defer func() { findSiteFn, syncEnvFn, syncFrontendFn = origFind, origSync, origFront }()

	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name}, nil
	}
	syncEnvFn = func(path, domain string, secured bool) error { return nil }

	called := false
	syncFrontendFn = func(path, domain string, secured bool) error { called = true; return nil }

	p := config.Proxy{
		Domains: []string{"x.localhost"}, UpstreamPort: 9000, Secured: true,
		Routes: []config.Route{{Path: "/api", Site: "api"}},
	}
	if err := syncProxyEnv(p); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Error("syncFrontendFn não deveria ser chamado sem p.Path")
	}
}

func TestRemove_RevertsFrontend(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	origFind, origSync, origReload, origUnsec := findSiteFn, syncEnvFn, nginxReloadFn, unsecureCertFn
	origFront, origRevert, origSecure := syncFrontendFn, revertFrontendFn, secureCertFn
	defer func() {
		findSiteFn, syncEnvFn, nginxReloadFn, unsecureCertFn = origFind, origSync, origReload, origUnsec
		syncFrontendFn, revertFrontendFn, secureCertFn = origFront, origRevert, origSecure
	}()
	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name, Domains: []string{name + ".localhost"}}, nil
	}
	syncEnvFn = func(path, domain string, secured bool) error { return nil }
	syncFrontendFn = func(path, domain string, secured bool) error { return nil }
	nginxReloadFn = func() error { return nil }
	unsecureCertFn = func(p config.Proxy) error { return nil }
	secureCertFn = func(p config.Proxy) error { return nil }

	var reverted string
	revertFrontendFn = func(path string) error { reverted = path; return nil }

	spaDir := t.TempDir()
	p, err := Add(AddOptions{
		Domain: "gc.localhost", Port: 9000, Path: spaDir,
		Routes: []config.Route{{Path: "/api", Site: "gc-api"}},
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := Remove(p.Name); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if reverted != p.Path {
		t.Errorf("reverted = %q, want %q", reverted, p.Path)
	}
}

func TestUpdate_RevertsFrontendOnPathClear(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	origFind, origSync, origReload := findSiteFn, syncEnvFn, nginxReloadFn
	origFront, origRevert, origSecure := syncFrontendFn, revertFrontendFn, secureCertFn
	defer func() {
		findSiteFn, syncEnvFn, nginxReloadFn = origFind, origSync, origReload
		syncFrontendFn, revertFrontendFn, secureCertFn = origFront, origRevert, origSecure
	}()
	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name, Domains: []string{name + ".localhost"}}, nil
	}
	syncEnvFn = func(path, domain string, secured bool) error { return nil }
	syncFrontendFn = func(path, domain string, secured bool) error { return nil }
	nginxReloadFn = func() error { return nil }
	secureCertFn = func(p config.Proxy) error { return nil }

	var reverted string
	revertFrontendFn = func(path string) error { reverted = path; return nil }

	spaDir := t.TempDir()
	p, err := Add(AddOptions{
		Domain: "gc.localhost", Port: 9000, Path: spaDir,
		Routes: []config.Route{{Path: "/api", Site: "gc-api"}},
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	empty := ""
	if _, err := Update(p.Name, UpdateOptions{Path: &empty}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if reverted != spaDir {
		t.Errorf("reverted = %q, want %q", reverted, spaDir)
	}
}
