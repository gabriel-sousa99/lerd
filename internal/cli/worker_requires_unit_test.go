//go:build linux

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// A worker naming a service it needs has to be ordered after it, or a boot
// starts the two together and the worker crash-loops until the service is up.
func TestRequiredServiceUnit_followsTheEnvCondition(t *testing.T) {
	site := t.TempDir()
	w := config.FrameworkWorker{
		RequiresService: &config.WorkerService{Name: "redis", WhenEnv: "QUEUE_CONNECTION=redis"},
	}

	if err := os.WriteFile(filepath.Join(site, ".env"), []byte("QUEUE_CONNECTION=database\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := requiredServiceUnit(site, w); got != "" {
		t.Errorf("database queue: got %q, want no ordering", got)
	}

	if err := os.WriteFile(filepath.Join(site, ".env"), []byte("QUEUE_CONNECTION=redis\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := requiredServiceUnit(site, w); got != "lerd-redis" {
		t.Errorf("redis queue: got %q, want %q", got, "lerd-redis")
	}

	if got := requiredServiceUnit(site, config.FrameworkWorker{}); got != "" {
		t.Errorf("worker declaring nothing: got %q, want no ordering", got)
	}
}

func TestWriteWorkerUnitFile_ordersAfterTheRequiredService(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	if _, err := writeWorkerUnitFile("lerd-queue-demo", "Queue Worker", "demo", t.TempDir(), "8.5",
		"php artisan queue:work", "always", "", "lerd-php85-fpm", "lerd-redis", false); err != nil {
		t.Fatal(err)
	}
	unit, err := os.ReadFile(filepath.Join(dir, "systemd", "user", "lerd-queue-demo.service"))
	if err != nil {
		t.Fatal(err)
	}
	body := string(unit)
	if !strings.Contains(body, "After=lerd-redis.service") || !strings.Contains(body, "Wants=lerd-redis.service") {
		t.Errorf("unit does not order after the service it needs:\n%s", body)
	}
}
