package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// NewCheckCmd returns the check command. Validating .lerd.yaml is now one check
// inside the site doctor, so this stays only as an alias for the muscle memory
// and points the user at the command that answers the whole question.
func NewCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "check",
		Short:        "Validate .lerd.yaml (alias for lerd site:doctor)",
		Deprecated:   "use `lerd site:doctor`, which validates .lerd.yaml as part of the site's health report.",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			if err := requireProjectConfig(cwd); err != nil {
				return err
			}
			return runSiteDoctor("", false, false)
		},
	}
}

// requireProjectConfig keeps the alias failing where the old command did. The
// site doctor answers for a directory whether or not it holds a project, so
// without this a run one directory too high reports that all checks pass.
func requireProjectConfig(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, ".lerd.yaml")); err != nil {
		return fmt.Errorf("no .lerd.yaml found in %s\nRun 'lerd init' to create one", dir)
	}
	return nil
}
