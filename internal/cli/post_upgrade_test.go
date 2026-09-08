package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// A package manager swaps the binary and runs nothing else, so the first
// command after the swap is what has to reapply the environment.
func TestShouldApplyUpgradeOnAVersionItHasNotSetUpYet(t *testing.T) {
	if !shouldApplyUpgrade("php", "1.32.0", "1.31.0", true, true) {
		t.Error("a set-up install running a newer binary should reapply the environment")
	}
}

func TestShouldApplyUpgradeSkipsWhenTheVersionIsUnchanged(t *testing.T) {
	if shouldApplyUpgrade("php", "1.32.0", "1.32.0", true, true) {
		t.Error("nothing changed, so nothing should run")
	}
}

// A fresh package install has a binary and no environment. The install is the
// user's own next step, and running it behind their back on the first command
// would take the choices that step asks them.
func TestShouldApplyUpgradeSkipsAMachineThatWasNeverSetUp(t *testing.T) {
	if shouldApplyUpgrade("php", "1.32.0", "", false, true) {
		t.Error("a machine with no lerd install should be left to `lerd install`")
	}
}

// Every install predating the stamp has no recorded version, so the upgrade
// that introduces it is the one that reapplies the environment once.
func TestShouldApplyUpgradeOnAnInstallFromBeforeTheStamp(t *testing.T) {
	if !shouldApplyUpgrade("php", "1.32.0", "", true, true) {
		t.Error("a set-up install with no recorded version should reapply once")
	}
}

// Reapplying restarts services and can take minutes, which is not something to
// spring on a script, a daemon or a pipe.
func TestShouldApplyUpgradeSkipsWithoutATerminal(t *testing.T) {
	if shouldApplyUpgrade("php", "1.32.0", "1.31.0", true, false) {
		t.Error("a non-interactive invocation should not start an install")
	}
}

// The install is what the reapply runs, so triggering from it would recurse,
// and the commands that already reapply have no need of a second pass.
func TestShouldApplyUpgradeSkipsTheCommandsThatRunItThemselves(t *testing.T) {
	for _, name := range []string{"install", "update", "bootstrap", "uninstall", "serve-ui", "watch", "mcp"} {
		if shouldApplyUpgrade(name, "1.32.0", "1.31.0", true, true) {
			t.Errorf("%s should not trigger the reapply", name)
		}
	}
}

// A build from a checkout carries a git describe version that changes with
// every commit, so honouring it would reinstall the environment all day.
func TestShouldApplyUpgradeSkipsDevelopmentBuilds(t *testing.T) {
	for _, v := range []string{"1.32.0-12-g5ca3460", "1.32.0-12-g5ca3460-dirty", "", "dev"} {
		if shouldApplyUpgrade("php", v, "1.31.0", true, true) {
			t.Errorf("version %q should not trigger the reapply", v)
		}
	}
}

// A release the user can actually be running, betas included, does trigger.
func TestIsReleaseVersionAcceptsReleases(t *testing.T) {
	for _, v := range []string{"1.32.0", "1.33.0-beta.1"} {
		if !isReleaseVersion(v) {
			t.Errorf("isReleaseVersion(%q) = false; want true", v)
		}
	}
}

// The skip list is written in the words the user types, so a subcommand has to
// be judged by the command it belongs to rather than its own leaf name.
func TestTopLevelCommandNamesTheGroup(t *testing.T) {
	root := &cobra.Command{Use: "lerd"}
	group := &cobra.Command{Use: "service"}
	leaf := &cobra.Command{Use: "start"}
	group.AddCommand(leaf)
	root.AddCommand(group)

	if got := topLevelCommand(leaf); got != "service" {
		t.Errorf("topLevelCommand() = %q; want %q", got, "service")
	}
	if got := topLevelCommand(group); got != "service" {
		t.Errorf("topLevelCommand() = %q; want %q", got, "service")
	}
}

func TestInstalledVersionRoundTrips(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	if got := readInstalledVersion(); got != "" {
		t.Errorf("readInstalledVersion() = %q on a fresh machine; want empty", got)
	}
	writeInstalledVersion("1.32.0")
	if got := readInstalledVersion(); got != "1.32.0" {
		t.Errorf("readInstalledVersion() = %q; want %q", got, "1.32.0")
	}
}

// The stamp records that the environment step ran, so a half-applied upgrade
// has to leave it alone and try again on the next command.
func TestClearInstalledVersionLetsAFailedReapplyRetry(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	writeInstalledVersion("1.32.0")

	clearInstalledVersion()

	if got := readInstalledVersion(); got != "" {
		t.Errorf("readInstalledVersion() = %q after clearing; want empty", got)
	}
}

func TestIsSetUpFollowsTheGlobalConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if isSetUp() {
		t.Error("isSetUp() = true with no global config")
	}
	if err := os.MkdirAll(filepath.Dir(config.GlobalConfigFile()), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.GlobalConfigFile(), []byte("dns:\n  tld: test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if !isSetUp() {
		t.Error("isSetUp() = false with a global config on disk")
	}
}

// `lerd install` shells out to itself for php:rebuild, and the version stamp is
// only written once the install finishes, so without this guard the child reads
// the install it is part of as a pending upgrade and reapplies the whole
// environment from inside it, rebuilding every PHP image twice over.
func TestShouldApplyUpgradeSkipsAChildOfARunningInstall(t *testing.T) {
	t.Setenv(installInProgressEnv, "1")
	if shouldApplyUpgrade("php:rebuild", "1.32.0", "1.31.0", true, true) {
		t.Error("a process spawned by a running install should not reapply the environment")
	}
}

func TestMarkInstallInProgressIsInheritedByChildren(t *testing.T) {
	t.Setenv(installInProgressEnv, "")
	markInstallInProgress()
	if os.Getenv(installInProgressEnv) == "" {
		t.Error("the install should mark its process tree so children skip the reapply")
	}
}
