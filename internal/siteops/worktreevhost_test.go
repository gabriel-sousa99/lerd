package siteops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func writeConf(t *testing.T, name string) string {
	t.Helper()
	if err := os.MkdirAll(config.NginxConfD(), 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(config.NginxConfD(), name)
	if err := os.WriteFile(p, []byte("server {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// A site's worktree subdomains are served by their own vhosts, so tearing the
// site down has to take them with it; otherwise nginx keeps routing every
// branch of a project that no longer exists.
func TestRemoveWorktreeVhosts_dropsBranchVhostsButNotTheSiteOrOtherSites(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	site := writeConf(t, "acme.test.conf")
	feature := writeConf(t, "feat-login.acme.test.conf")
	release := writeConf(t, "release-2.acme.test.conf")
	other := writeConf(t, "shop.test.conf")

	removed := RemoveWorktreeVhosts("acme.test")

	if len(removed) != 2 {
		t.Errorf("removed = %v, want both branch domains", removed)
	}
	for _, p := range []string{feature, release} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s survived", filepath.Base(p))
		}
	}
	for _, p := range []string{site, other} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s must not be touched: %v", filepath.Base(p), err)
		}
	}
}

// A separately registered site can live at a subdomain of another one (a group
// secondary at admin.acme.test). Its vhost matches the same suffix scan, and
// deleting it would stop serving a site that is still linked.
func TestRemoveWorktreeVhosts_keepsARegisteredSubdomainSite(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if err := config.SaveSites(&config.SiteRegistry{Sites: []config.Site{
		{Name: "admin", Domains: []string{"admin.acme.test"}, Path: t.TempDir()},
	}}); err != nil {
		t.Fatal(err)
	}
	admin := writeConf(t, "admin.acme.test.conf")
	branch := writeConf(t, "feat-login.acme.test.conf")

	RemoveWorktreeVhosts("acme.test")

	if _, err := os.Stat(admin); err != nil {
		t.Errorf("a registered subdomain site lost its vhost: %v", err)
	}
	if _, err := os.Stat(branch); !os.IsNotExist(err) {
		t.Error("the worktree vhost survived")
	}
}
