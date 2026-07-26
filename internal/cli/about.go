package cli

import (
	"fmt"

	"github.com/gabriel-sousa99/lerd/internal/feedback"
	"github.com/gabriel-sousa99/lerd/internal/version"
	"github.com/spf13/cobra"
)

// NewAboutCmd returns the about command.
func NewAboutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "about",
		Short: "Show information about Lerd",
		RunE:  runAbout,
	}
}

func runAbout(_ *cobra.Command, _ []string) error {
	feedback.Begin()
	fmt.Println("  " + feedback.Title("lerd Oracle Edition"))
	fmt.Println("  " + feedback.Dim("Podman-powered local PHP development with baked-in Oracle Database support"))
	feedback.NewSummary().
		Row("Version", feedback.Val(version.Version)).
		Row("Commit", version.Commit).
		Row("Built", version.Date).
		Row("Fork", feedback.Val("https://github.com/gabriel-sousa99/lerd")).
		Row("Oracle", "Instant Client 21.18 + oci8").
		Row("Upstream", "https://github.com/lerd-env/lerd").
		Print()
	feedback.Begin()
	fmt.Println("  " + feedback.Dim("Oracle additions by Gabriel Sousa — built on lerd, © George Dumitrescu"))
	return nil
}
