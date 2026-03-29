package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mirurobotics/gotools/internal/covgate"
	"github.com/mirurobotics/gotools/internal/coverage"
	"github.com/mirurobotics/gotools/internal/covratchet"
	"github.com/mirurobotics/gotools/internal/deps"
	"github.com/mirurobotics/gotools/internal/gotest"
	"github.com/mirurobotics/gotools/internal/lint"
	"github.com/mirurobotics/gotools/internal/tag"
)

func main() {
	root := &cobra.Command{
		Use:   "miru",
		Short: "Miru Go development tools",
		Long:  "A collection of Go development tools for Miru projects.",
	}

	root.AddCommand(lint.NewLintCommand())
	root.AddCommand(lint.NewCommand())
	root.AddCommand(covgate.NewCommand())
	root.AddCommand(covratchet.NewCommand())
	root.AddCommand(coverage.NewCommand())
	root.AddCommand(gotest.NewCommand())
	root.AddCommand(deps.NewCommand())
	root.AddCommand(tag.NewCommand())

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
