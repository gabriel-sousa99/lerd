package proxyops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestSetSecuredToggle(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	prevS := secureCertFn
	prevU := unsecureCertFn
	secureCertFn = func(config.Proxy) error { return nil }
	unsecureCertFn = func(config.Proxy) error { return nil }
	defer func() { secureCertFn = prevS; unsecureCertFn = prevU }()

	_, err := Add(AddOptions{Domain: "spa.localhost", Port: 9000, NoSecure: true, Path: dir})
	if err != nil {
		t.Fatal(err)
	}

	p, _ := config.FindProxy("spa")
	if err := SetSecured(p, true); err != nil {
		t.Fatalf("SetSecured(true): %v", err)
	}
	got, _ := config.FindProxy("spa")
	if !got.Secured {
		t.Fatalf("expected Secured=true")
	}
	if _, err := os.Stat(filepath.Join(config.NginxConfD(), "spa.localhost-ssl.conf")); err != nil {
		t.Fatalf("expected ssl vhost: %v", err)
	}

	if err := SetSecured(got, false); err != nil {
		t.Fatalf("SetSecured(false): %v", err)
	}
	got2, _ := config.FindProxy("spa")
	if got2.Secured {
		t.Fatalf("expected Secured=false")
	}
	if _, err := os.Stat(filepath.Join(config.NginxConfD(), "spa.localhost.conf")); err != nil {
		t.Fatalf("expected plain vhost: %v", err)
	}
}
