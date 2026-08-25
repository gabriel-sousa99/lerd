package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gabriel-sousa99/lerd/internal/services"
)

// Seams for the hook's effects, swapped in tests so the decision can be checked
// without a service manager or a real install underneath.
var (
	healUpgradedBinary = healLerdBinaryUpgrade
	daemonEnabled      = func(name string) bool { return services.Mgr.IsEnabled(name) }
	restartDaemon      = func(name string) error { return services.Mgr.Restart(name) }
)

// NewPostUpgradeCmd returns the hook a package manager runs once it has put a
// new lerd binary in place. Homebrew's post-install step calls it, because an
// upgrade that lands unattended otherwise leaves the daemons dead and `php`
// broken until someone opens a terminal and runs a lerd command (#1432).
func NewPostUpgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "post-upgrade",
		Short:  "Repoint lerd's user services and shims at the binary installed now",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			runPostUpgradeHook()
			return nil
		},
	}
}

// runPostUpgradeHook repairs what replacing the binary broke, and nothing else.
// The rest of the environment (quadlets, nginx, the stores) is reapplied by the
// first lerd command run at a terminal, which has someone watching and can
// afford the minutes. It never fails: a package manager whose post-install step
// exits non-zero reports the upgrade itself as broken.
func runPostUpgradeHook() {
	// A machine with no install has nothing pointing anywhere yet, and the
	// `lerd install` it still owes asks questions a package script cannot
	// answer on the user's behalf.
	if !isSetUp() {
		return
	}
	units, shims := healUpgradedBinary()
	if len(units)+len(shims) == 0 {
		return
	}
	fmt.Println(repairSummary(units, shims))
	for _, name := range units {
		if !daemonEnabled(name) {
			continue
		}
		if err := restartDaemon(name); err != nil {
			fmt.Printf("  WARN: restarting %s: %v\n", name, err)
		}
	}
}
