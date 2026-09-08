package nginx

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// setXdebugMode puts a mode on the version in the global config, the way
// `lerd xdebug` does.
func setXdebugMode(t *testing.T, version, mode string) {
	t.Helper()
	cfg, err := config.LoadGlobal()
	if err != nil {
		t.Fatalf("LoadGlobal: %v", err)
	}
	if cfg.PHP.XdebugMode == nil {
		cfg.PHP.XdebugMode = map[string]string{}
	}
	cfg.PHP.XdebugMode[version] = mode
	if err := config.SaveGlobal(cfg); err != nil {
		t.Fatalf("SaveGlobal: %v", err)
	}
}

// A request paused at a breakpoint sends nginx nothing, so the 60s default
// turns every debugging session into a 504 while the session is still alive.
func TestGenerateVhost_liftsTimeoutWhenXdebugDebugs(t *testing.T) {
	confD := setupConfD(t)
	setXdebugMode(t, "8.4", "debug")

	site := config.Site{Name: "app", Domains: []string{"app.test"}, Path: t.TempDir()}
	if err := GenerateVhost(site, "8.4"); err != nil {
		t.Fatalf("GenerateVhost: %v", err)
	}
	content := readConf(t, filepath.Join(confD, "app.test.conf"))
	want := "fastcgi_read_timeout 3600s;"
	if !strings.Contains(content, want) {
		t.Errorf("expected %q in:\n%s", want, content)
	}
}

// A combined mode still debugs, so it still needs the room.
func TestGenerateSSLVhost_liftsTimeoutForCombinedMode(t *testing.T) {
	confD := setupConfD(t)
	setXdebugMode(t, "8.3", "develop,debug")

	site := config.Site{Name: "app", Domains: []string{"app.test"}, Path: t.TempDir()}
	if err := GenerateSSLVhost(site, "8.3"); err != nil {
		t.Fatalf("GenerateSSLVhost: %v", err)
	}
	content := readConf(t, filepath.Join(confD, "app.test-ssl.conf"))
	if !strings.Contains(content, "fastcgi_read_timeout 3600s;") {
		t.Errorf("expected the lifted timeout in:\n%s", content)
	}
}

// Xdebug loaded but not debugging changes nothing.
func TestGenerateVhost_keepsDefaultWhenXdebugOff(t *testing.T) {
	confD := setupConfD(t)
	setXdebugMode(t, "8.4", "off")

	site := config.Site{Name: "app", Domains: []string{"app.test"}, Path: t.TempDir()}
	if err := GenerateVhost(site, "8.4"); err != nil {
		t.Fatalf("GenerateVhost: %v", err)
	}
	content := readConf(t, filepath.Join(confD, "app.test.conf"))
	if !strings.Contains(content, "fastcgi_read_timeout 60s;") {
		t.Errorf("expected the 60s default in:\n%s", content)
	}
}

// Another version debugging says nothing about this site's.
func TestGenerateVhost_ignoresAnotherVersionsMode(t *testing.T) {
	confD := setupConfD(t)
	setXdebugMode(t, "8.3", "debug")

	site := config.Site{Name: "app", Domains: []string{"app.test"}, Path: t.TempDir()}
	if err := GenerateVhost(site, "8.4"); err != nil {
		t.Fatalf("GenerateVhost: %v", err)
	}
	content := readConf(t, filepath.Join(confD, "app.test.conf"))
	if !strings.Contains(content, "fastcgi_read_timeout 60s;") {
		t.Errorf("expected the 60s default in:\n%s", content)
	}
}

// An explicit project timeout is a deliberate answer, so it still wins.
func TestGenerateVhost_projectTimeoutWinsOverXdebugLift(t *testing.T) {
	confD := setupConfD(t)
	setXdebugMode(t, "8.4", "debug")

	projectDir := t.TempDir()
	if err := config.SaveProjectConfig(projectDir, &config.ProjectConfig{RequestTimeout: 120}); err != nil {
		t.Fatalf("SaveProjectConfig: %v", err)
	}
	site := config.Site{Name: "app", Domains: []string{"app.test"}, Path: projectDir}
	if err := GenerateVhost(site, "8.4"); err != nil {
		t.Fatalf("GenerateVhost: %v", err)
	}
	content := readConf(t, filepath.Join(confD, "app.test.conf"))
	if !strings.Contains(content, "fastcgi_read_timeout 120s;") {
		t.Errorf("expected the project's 120s in:\n%s", content)
	}
}
