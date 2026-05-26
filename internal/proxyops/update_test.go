package proxyops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }

func TestUpdateChangesPortRegeneratesVhost(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	StubForTests()
	defer UnstubForTests()

	if _, err := Add(AddOptions{Domain: "spa.localhost", Port: 9000, NoSecure: true, Path: dir}); err != nil {
		t.Fatal(err)
	}

	// Remove the existing vhost so we can assert it gets rewritten.
	vhost := filepath.Join(config.NginxConfD(), "spa.localhost.conf")
	if err := os.Remove(vhost); err != nil {
		t.Fatalf("remove vhost: %v", err)
	}

	updated, err := Update("spa", UpdateOptions{Port: intPtr(9001)})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.UpstreamPort != 9001 {
		t.Fatalf("port: got %d want 9001", updated.UpstreamPort)
	}
	if _, err := os.Stat(vhost); err != nil {
		t.Fatalf("vhost not regenerated: %v", err)
	}

	reg, _ := config.LoadProxies()
	if reg.Proxies[0].UpstreamPort != 9001 {
		t.Fatalf("registry not persisted: %d", reg.Proxies[0].UpstreamPort)
	}
}

func TestUpdateNoChangeSkipsVhost(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	StubForTests()
	defer UnstubForTests()

	if _, err := Add(AddOptions{Domain: "spa.localhost", Port: 9000, NoSecure: true, Path: dir}); err != nil {
		t.Fatal(err)
	}
	vhost := filepath.Join(config.NginxConfD(), "spa.localhost.conf")
	if err := os.Remove(vhost); err != nil {
		t.Fatalf("remove vhost: %v", err)
	}

	// Update with no fields — vhost must NOT be regenerated.
	if _, err := Update("spa", UpdateOptions{}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, err := os.Stat(vhost); err == nil {
		t.Fatalf("vhost was regenerated even though nothing changed")
	}
}

func TestUpdateRejectsInvalidPort(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	StubForTests()
	defer UnstubForTests()

	if _, err := Add(AddOptions{Domain: "spa.localhost", Port: 9000, NoSecure: true, Path: dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := Update("spa", UpdateOptions{Port: intPtr(70000)}); err == nil {
		t.Fatalf("expected invalid port error")
	}
	if _, err := Update("spa", UpdateOptions{Port: intPtr(0)}); err == nil {
		t.Fatalf("expected invalid port error (zero)")
	}
}

func TestUpdateRejectsMissingPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	StubForTests()
	defer UnstubForTests()

	if _, err := Add(AddOptions{Domain: "spa.localhost", Port: 9000, NoSecure: true, Path: dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := Update("spa", UpdateOptions{Path: strPtr("/nope/missing/path")}); err == nil {
		t.Fatalf("expected path error")
	}
}

func TestUpdateClearsPathOnNonManaged(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	StubForTests()
	defer UnstubForTests()

	if _, err := Add(AddOptions{Domain: "spa.localhost", Port: 9000, NoSecure: true, Path: dir}); err != nil {
		t.Fatal(err)
	}
	updated, err := Update("spa", UpdateOptions{Path: strPtr("")})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Path != "" {
		t.Fatalf("path: got %q want empty", updated.Path)
	}
}

func TestUpdateMissingProxy(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	StubForTests()
	defer UnstubForTests()

	if _, err := Update("ghost", UpdateOptions{Port: intPtr(9001)}); err == nil {
		t.Fatalf("expected not-found error")
	}
}

func TestUpdateSkipsVhostWhenPaused(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	StubForTests()
	defer UnstubForTests()

	if _, err := Add(AddOptions{Domain: "spa.localhost", Port: 9000, NoSecure: true, Path: dir}); err != nil {
		t.Fatal(err)
	}
	// Mark paused via the registry directly to avoid pulling in ApplyPause's nginx side effects.
	p, _ := config.FindProxy("spa")
	p.Paused = true
	_ = config.AddProxy(*p)

	vhost := filepath.Join(config.NginxConfD(), "spa.localhost.conf")
	_ = os.Remove(vhost)

	if _, err := Update("spa", UpdateOptions{Port: intPtr(9001)}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, err := os.Stat(vhost); err == nil {
		t.Fatalf("vhost regenerated for a paused proxy")
	}
	reg, _ := config.LoadProxies()
	if reg.Proxies[0].UpstreamPort != 9001 {
		t.Fatalf("registry not updated: %d", reg.Proxies[0].UpstreamPort)
	}
}
