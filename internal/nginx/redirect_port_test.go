package nginx

import (
	"os"
	"path/filepath"
	"testing"
)

// writeNginxPortConfig points the global config at a temp dir holding the given
// nginx port block, so the helper reads a configured value without the real one.
func writeNginxPortConfig(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Join(dir, "lerd"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lerd", "config.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A host that moved nginx off 443 must be redirected to the port it moved to,
// or every plain-HTTP visit lands on a port nothing listens on.
func TestHTTPSRedirectHost_carriesAMovedPort(t *testing.T) {
	writeNginxPortConfig(t, "nginx:\n    http_port: 10080\n    https_port: 10443\n")
	if got := (VhostData{}).HTTPSRedirectHost(); got != "$host:10443" {
		t.Errorf("HTTPSRedirectHost = %q, want %q", got, "$host:10443")
	}
}

func TestHTTPSRedirectHost_staysBareOnTheDefaultPort(t *testing.T) {
	writeNginxPortConfig(t, "nginx:\n    http_port: 80\n    https_port: 443\n")
	if got := (VhostData{}).HTTPSRedirectHost(); got != "$host" {
		t.Errorf("HTTPSRedirectHost = %q, want %q", got, "$host")
	}
}
