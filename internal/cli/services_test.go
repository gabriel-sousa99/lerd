package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"

	"github.com/spf13/cobra"
)

func TestServiceReinstallCmd_HasResetDataFlag(t *testing.T) {
	cmd := newServiceReinstallCmd()
	flag := cmd.Flags().Lookup("reset-data")
	if flag == nil {
		t.Fatal("--reset-data flag missing on `service reinstall`")
	}
	if flag.Value.Type() != "bool" {
		t.Errorf("--reset-data flag type = %q, want bool", flag.Value.Type())
	}
}

func TestServiceReinstallCmd_WiredIntoServiceCmd(t *testing.T) {
	parent := NewServiceCmd()
	var found bool
	for _, c := range parent.Commands() {
		if c.Name() == "reinstall" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("`service reinstall` subcommand not registered under `service`")
	}
}

func TestServiceRemoveCmd_HasPurgeFlag(t *testing.T) {
	cmd := newServiceRemoveCmd()
	flag := cmd.Flags().Lookup("purge")
	if flag == nil {
		t.Fatal("--purge flag missing on `service remove`")
	}
	if flag.Value.Type() != "bool" {
		t.Errorf("--purge flag type = %q, want bool", flag.Value.Type())
	}
	if flag.DefValue != "false" {
		t.Errorf("--purge default = %q, want false", flag.DefValue)
	}
}

// Both data-wiping commands snapshot the databases first, so both need the
// same escape hatch for an engine that cannot be brought up to dump from.
func TestDataWipingCommandsHaveNoSnapshotFlag(t *testing.T) {
	for _, tc := range []struct {
		use string
		cmd func() *cobra.Command
	}{
		{"service remove", newServiceRemoveCmd},
		{"service reinstall", newServiceReinstallCmd},
	} {
		flag := tc.cmd().Flags().Lookup("no-snapshot")
		if flag == nil {
			t.Errorf("--no-snapshot flag missing on `%s`", tc.use)
			continue
		}
		if flag.DefValue != "false" {
			t.Errorf("`%s` --no-snapshot default = %q, want false", tc.use, flag.DefValue)
		}
	}
}

// migrateServiceUnits must only refresh quadlets that already exist on disk.
// If a default service was deliberately removed (no quadlet present), the
// next CLI invocation must NOT silently recreate it.
func TestMigrateServiceUnits_SkipsUninstalledDefaultPresets(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)

	quadletDir := config.QuadletDir()
	if err := os.MkdirAll(quadletDir, 0o755); err != nil {
		t.Fatalf("mkdir quadlet dir: %v", err)
	}

	// Sanity: no quadlets exist for any default preset in this fresh tmp env.
	migrateServiceUnits()

	for _, svc := range knownServices() {
		path := filepath.Join(quadletDir, "lerd-"+svc+".container")
		if _, err := os.Stat(path); err == nil {
			t.Errorf("quadlet for uninstalled default preset %q was recreated at %s", svc, path)
		}
	}
}
