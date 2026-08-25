package cli

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// Unlinking a site releases its shares, and a worktree's share is the site's
// too: it is keyed per branch, so a site-only release leaves LAN proxies bound
// and registry entries pointing at a project that is gone.
func TestStopSiteShares_releasesTheWorktreeShares(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	for _, e := range []config.WorktreeLANEntry{
		{Site: "acme", Branch: "feat-login", Port: 9101},
		{Site: "acme", Branch: "release-2", Port: 9102},
		{Site: "acme-shop", Branch: "feat-cart", Port: 9103},
	} {
		if err := config.AddWorktreeLAN(e); err != nil {
			t.Fatal(err)
		}
	}

	stopSiteShares("acme")

	left, err := config.WorktreeLANsForSite("acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Errorf("the unlinked site keeps %d worktree LAN shares", len(left))
	}
	other, err := config.WorktreeLANsForSite("acme-shop")
	if err != nil {
		t.Fatal(err)
	}
	if len(other) != 1 {
		t.Errorf("a site whose name merely starts the same lost its share: %v", other)
	}
}

func TestTunnelKeyBelongsToSite(t *testing.T) {
	cases := []struct {
		key, site string
		want      bool
	}{
		{"acme", "acme", true},
		{"acme@feat-login", "acme", true},
		{"acme-shop", "acme", false},
		{"acme-shop@feat-cart", "acme", false},
		{"other", "acme", false},
	}
	for _, c := range cases {
		if got := tunnelKeyBelongsToSite(c.key, c.site); got != c.want {
			t.Errorf("tunnelKeyBelongsToSite(%q, %q) = %v, want %v", c.key, c.site, got, c.want)
		}
	}
}
