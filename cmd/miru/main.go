package main

import (
	"fmt"
	"os"

	"github.com/mirurobotics/gotools/internal/commands"
)

func main() {
	if err := commands.NewRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
