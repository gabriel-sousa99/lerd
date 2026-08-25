package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// The daemons are dead the moment the binary they name is replaced, so the hook
// that repointed their units has to start them again. Nothing else is watching:
// this is the upgrade that landed while the user was away.
func TestPostUpgradeHookRestartsTheDaemonsItRepointed(t *testing.T) {
	markSetUp(t)
	hookFixture(t, []string{"lerd-ui", "lerd-watcher"}, []string{"php"})
	enabled := map[string]bool{"lerd-ui": true, "lerd-watcher": true}
	daemonEnabled = func(name string) bool { return enabled[name] }
	var restarted []string
	restartDaemon = func(name string) error {
		restarted = append(restarted, name)
		return nil
	}

	runPostUpgradeHook()

	if !equalStrings(restarted, []string{"lerd-ui", "lerd-watcher"}) {
		t.Errorf("restarted %v; want both repointed daemons started again", restarted)
	}
}

// A daemon the user never wanted at login stays down. The tray is the case:
// on a headless machine it is installed and disabled, and starting it would
// fail for a reason that has nothing to do with the upgrade.
func TestPostUpgradeHookLeavesDisabledDaemonsDown(t *testing.T) {
	markSetUp(t)
	hookFixture(t, []string{"lerd-ui", "lerd-tray"}, nil)
	daemonEnabled = func(name string) bool { return name == "lerd-ui" }
	var restarted []string
	restartDaemon = func(name string) error {
		restarted = append(restarted, name)
		return nil
	}

	runPostUpgradeHook()

	if !equalStrings(restarted, []string{"lerd-ui"}) {
		t.Errorf("restarted %v; want only the daemon set to start at login", restarted)
	}
}

// A fresh package install has a binary and no environment. The install it still
// owes is the user's own next step, and it asks questions.
func TestPostUpgradeHookSkipsAMachineThatWasNeverSetUp(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	hookFixture(t, []string{"lerd-ui"}, nil)
	healed := false
	healUpgradedBinary = func() ([]string, []string) {
		healed = true
		return nil, nil
	}
	restartDaemon = func(string) error {
		t.Error("a machine with no lerd install had a daemon restarted")
		return nil
	}

	runPostUpgradeHook()

	if healed {
		t.Error("a machine with no lerd install was repaired behind the user's back")
	}
}

// An upgrade that broke nothing, which is every upgrade of an install that
// carries the version-independent path already, must not restart anything.
func TestPostUpgradeHookIsQuietWhenNothingMoved(t *testing.T) {
	markSetUp(t)
	hookFixture(t, nil, nil)
	restartDaemon = func(name string) error {
		t.Errorf("%s was restarted with nothing to repair", name)
		return nil
	}

	runPostUpgradeHook()
}

// A shim repaired on its own is worth reporting, but there is no daemon to
// start for it.
func TestPostUpgradeHookRestartsNothingForShimsAlone(t *testing.T) {
	markSetUp(t)
	hookFixture(t, nil, []string{"php", "composer"})
	restartDaemon = func(name string) error {
		t.Errorf("%s was restarted for a shim repair", name)
		return nil
	}

	runPostUpgradeHook()
}

// markSetUp writes the global config, which is what tells the hook an install
// exists to repair.
func markSetUp(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := os.MkdirAll(filepath.Dir(config.GlobalConfigFile()), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.GlobalConfigFile(), []byte("dns:\n  tld: test\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

// hookFixture makes the repair report the given units and shims, and restores
// every seam once the test is done.
func hookFixture(t *testing.T, units, shims []string) {
	t.Helper()
	t.Cleanup(restoreHookSeams(t))
	healUpgradedBinary = func() ([]string, []string) { return units, shims }
	daemonEnabled = func(string) bool { return true }
	restartDaemon = func(string) error { return nil }
}

func restoreHookSeams(t *testing.T) func() {
	t.Helper()
	heal, enabled, restart := healUpgradedBinary, daemonEnabled, restartDaemon
	return func() { healUpgradedBinary, daemonEnabled, restartDaemon = heal, enabled, restart }
}
