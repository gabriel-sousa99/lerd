package proxyops

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/reqstats"
)

func TestRemoveDeletesEverything(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	// Stub mkcert
	prevS := secureCertFn
	prevU := unsecureCertFn
	secureCertFn = func(config.Proxy) error { return nil }
	unsecureCertFn = func(config.Proxy) error { return nil }
	defer func() { secureCertFn = prevS; unsecureCertFn = prevU }()

	_, err := Add(AddOptions{Domain: "spa.localhost", Port: 9000, NoSecure: true, Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	key := reqstats.ProxyKey("spa")
	if err := reqstats.SaveSnapshot([]reqstats.SiteStats{{Site: key, Samples: 2, UpdatedAt: time.Now()}}, config.RequestStatsFile()); err != nil {
		t.Fatal(err)
	}
	if err := Remove("spa"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	reg, _ := config.LoadProxies()
	if len(reg.Proxies) != 0 {
		t.Fatalf("expected empty registry after remove, got %d", len(reg.Proxies))
	}
	conf := filepath.Join(config.NginxConfD(), "spa.localhost.conf")
	if _, err := os.Stat(conf); !os.IsNotExist(err) {
		t.Fatalf("vhost was not removed")
	}
	if _, ok := reqstats.LoadSite(config.RequestStatsFile(), key); ok {
		t.Fatal("proxy request stats were not removed")
	}
}
