package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/podman"
)

// A framework's own MCP server is started through lerd (`lerd artisan mcp:start`,
// or `php artisan …` via the host shim), so the child owns stdout for the
// JSON-RPC stream. Every setup notice lerd prints before the exec has to go to
// stderr, or the first line it writes desynchronises the client.

func TestEnsureServiceRunning_progressStaysOffStdout(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	out := captureStdout(t, func() { _ = ensureServiceRunning("totallycustom") })
	if out != "" {
		t.Errorf("stdout = %q, want empty (it belongs to the exec'd child)", out)
	}
}

// stoppedUnits reports every unit as inactive, so the paused-site notice is
// reached whatever is actually running on the machine running the test.
type stoppedUnits struct{}

func (stoppedUnits) Start(string) error                { return nil }
func (stoppedUnits) Stop(string) error                 { return nil }
func (stoppedUnits) Restart(string) error              { return nil }
func (stoppedUnits) UnitStatus(string) (string, error) { return "inactive", nil }
func (stoppedUnits) AllUnitStates() map[string]string  { return nil }

func TestStartServicesForSite_noticesStayOffStdout(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	prevUnits := podman.UnitLifecycle
	podman.UnitLifecycle = stoppedUnits{}
	t.Cleanup(func() { podman.UnitLifecycle = prevUnits })

	prev := ensureServiceRunningFn
	ensureServiceRunningFn = func(string) error { return os.ErrNotExist }
	t.Cleanup(func() { ensureServiceRunningFn = prev })

	dir := t.TempDir()
	env := "DB_HOST=lerd-mysql\nREDIS_HOST=lerd-redis\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(env), 0644); err != nil {
		t.Fatalf("writing .env: %v", err)
	}

	out := captureStdout(t, func() { startServicesForSiteNoticed(dir, "demo") })
	if out != "" {
		t.Errorf("stdout = %q, want empty (paused notice and warnings belong on stderr)", out)
	}
}
