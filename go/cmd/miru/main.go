package main

import (
	"fmt"
	"os"

	"github.com/mirurobotics/tools/go/internal/commands"
)

func main() {
	if err := commands.NewRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
