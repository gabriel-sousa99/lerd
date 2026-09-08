package nginx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// The whole point of nginx.http_port / https_port is moving nginx off :80/:443
// on a host where something else owns them, so the unit that actually binds has
// to carry the configured ports. It never did (#1544): both writers rendered the
// bundled template, which publishes the defaults, and nothing substituted them.
func TestRewriteNginxQuadlet_honoursConfiguredPorts(t *testing.T) {
	sandbox := t.TempDir()
	home := filepath.Join(sandbox, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(sandbox, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(sandbox, "data"))

	cfg := &config.GlobalConfig{}
	cfg.Nginx.HTTPPort = 10080
	cfg.Nginx.HTTPSPort = 10443
	if err := config.SaveGlobal(cfg); err != nil {
		t.Fatalf("saving config: %v", err)
	}

	if _, err := RewriteNginxQuadlet(); err != nil {
		t.Fatalf("RewriteNginxQuadlet: %v", err)
	}

	unit, err := os.ReadFile(filepath.Join(config.QuadletDir(), "lerd-nginx.container"))
	if err != nil {
		t.Fatalf("reading generated unit: %v", err)
	}
	got := string(unit)

	// The host side moves; the container side stays on 80/443 because that is
	// what nginx listens on inside the image. The written unit also carries the
	// loopback and IPv6 prefixes WriteQuadletDiff adds, so match the mapping
	// rather than the whole line.
	for _, want := range []string{":10080:80", ":10443:443"} {
		if !strings.Contains(got, want) {
			t.Errorf("generated unit missing a %q mapping:\n%s", want, got)
		}
	}
	if strings.Contains(got, ":80:80") || strings.Contains(got, ":443:443") {
		t.Errorf("generated unit still publishes the defaults:\n%s", got)
	}
}
