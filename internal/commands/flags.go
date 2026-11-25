package commands

import "github.com/spf13/pflag"

func bindLayoutFlags(flags *pflag.FlagSet, srcPrefix, testDir *string) {
	flags.StringVar(
		srcPrefix, "src-prefix", "pkg",
		"source prefix for mapping external test dirs",
	)
	flags.StringVar(testDir, "test-dir", "", "external test directory (empty = none)")
}
