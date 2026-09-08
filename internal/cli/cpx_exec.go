package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// NewCpxCmd returns the cpx command. cpx is to Composer what npx is to npm: it
// runs a command from any Composer package without adding it to the project.
// lerd runs the globally required binary inside the project's FPM container, so
// the package it fetches runs on the PHP version that project is registered on
// rather than whatever PHP happens to be on the host.
func NewCpxCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "cpx [args...]",
		Short:              "Run cpx (Composer's npx) in the project's container",
		DisableFlagParsing: true,
		SilenceUsage:       true,
		RunE: func(_ *cobra.Command, args []string) error {
			return runCpx(args)
		},
	}
}

// cpxBinPath is where `composer global require cpx/cpx` drops the binary.
func cpxBinPath() string {
	return filepath.Join(composerGlobalBinDir(), "cpx")
}

func runCpx(args []string) error {
	bin := cpxBinPath()
	if _, err := os.Stat(bin); err != nil {
		return fmt.Errorf("cpx is not installed: run `lerd composer global require cpx/cpx`")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	// cpx caches the packages it fetches under $HOME/.cpx, and lerd already
	// execs with HOME set to the host's home, so the cache is shared with the
	// host and survives a container restart. Nothing to pass here.
	code, runErr := RunPHPCaptureEnv(cwd, append([]string{bin}, args...), nil)
	if runErr != nil {
		return runErr
	}
	if code != 0 {
		os.Exit(code)
	}
	return nil
}
