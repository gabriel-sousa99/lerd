package proxyops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestAddProxyHappyPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	// Stub mkcert para não invocar binário.
	prev := secureCertFn
	secureCertFn = func(config.Proxy) error { return nil }
	defer func() { secureCertFn = prev }()

	p, err := Add(AddOptions{
		Domain: "spa.localhost",
		Port:   9000,
		Path:   dir, // existing dir
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if p.Name != "spa" {
		t.Fatalf("name: got %q want %q", p.Name, "spa")
	}
	if p.PrimaryDomain() != "spa.localhost" {
		t.Fatalf("domain: got %q", p.PrimaryDomain())
	}
	if !p.Secured {
		t.Fatalf("expected Secured=true by default")
	}

	confPath := filepath.Join(config.NginxConfD(), "spa.localhost-ssl.conf")
	if _, err := os.Stat(confPath); err != nil {
		t.Fatalf("missing vhost: %v", err)
	}
}

func TestAddProxyRejectsDuplicateDomain(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	if _, err := Add(AddOptions{Domain: "spa.localhost", Port: 9000, NoSecure: true, Path: dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(AddOptions{Domain: "spa.localhost", Port: 9001, NoSecure: true, Path: dir}); err == nil {
		t.Fatalf("expected dup error")
	}
}

func TestAddProxyRejectsConflictingSiteDomain(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)

	// Registra um Site com o mesmo domínio.
	_ = config.AddSite(config.Site{
		Name:    "myapp",
		Domains: []string{"myapp.localhost"},
		Path:    dir,
	})

	if _, err := Add(AddOptions{Domain: "myapp.localhost", Port: 9000, NoSecure: true, Path: dir}); err == nil {
		t.Fatalf("expected conflict with site domain")
	}
}
