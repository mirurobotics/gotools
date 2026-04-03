package commands

import (
	"fmt"
	"os"

	"github.com/mirurobotics/gotools/internal/services/huntdocs"

	"github.com/spf13/cobra"
)

// NewHuntDocsCommand returns a cobra command that scans Go
// source files for missing or incomplete documentation.
func NewHuntDocsCommand() *cobra.Command {
	var (
		path        string
		minSeverity string
	)

	//nolint:exhaustruct // cobra uses partial initialization
	cmd := &cobra.Command{
		Use:   "hunt-docs",
		Short: "Find missing Go documentation",
		Long: "Scan Go source files for missing or incomplete " +
			"documentation on exported symbols.\n\n" +
			"Reports undocumented packages, functions, types, " +
			"methods, constants, variables, and struct fields, " +
			"ranked by severity.",
		RunE: func(_ *cobra.Command, _ []string) error {
			sev, err := huntdocs.ParseSeverity(minSeverity)
			if err != nil {
				return err
			}

			gaps, err := huntdocs.Run(huntdocs.Opts{
				Path:        path,
				MinSeverity: sev,
				Out:         os.Stdout,
				Err:         os.Stderr,
			})
			if err != nil {
				return err
			}

			for _, g := range gaps {
				fmt.Fprintln(os.Stdout, g)
			}

			if len(gaps) == 0 {
				fmt.Fprintln(os.Stdout, "no documentation gaps found")
				return nil
			}

			fmt.Fprintf(os.Stdout, "\n%d documentation gap(s) found\n", len(gaps))
			return nil
		},
	}

	fl := cmd.Flags()
	fl.StringVar(&path, "path", ".", "directory to scan")
	fl.StringVar(
		&minSeverity, "min-severity", "high",
		"minimum severity to report (critical, high, medium)",
	)

	return cmd
}
