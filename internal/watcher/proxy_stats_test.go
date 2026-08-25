package watcher

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/reqstats"
)

func TestResolveHostToStatsKeyResolvesProxyDomains(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.AddProxy(config.Proxy{
		Name: "spa", Domains: []string{"spa.localhost", "admin.spa.localhost"}, UpstreamPort: 5173,
	}); err != nil {
		t.Fatal(err)
	}

	for _, host := range []string{"spa.localhost", "admin.spa.localhost"} {
		got, ok := resolveHostToStatsKey(host)
		if !ok || got != reqstats.ProxyKey("spa") {
			t.Fatalf("stats key for %s = %q (ok=%v), want %q", host, got, ok, reqstats.ProxyKey("spa"))
		}
	}
}
