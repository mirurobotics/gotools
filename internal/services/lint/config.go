package lint

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter"
)

// CheckOpts holds the options for the check command.
type CheckOpts struct {
	Path          string
	DoFix         bool
	MaxLineWidth  int
	TabWidth      int
	MaxFuncLen    int
	MaxNestDepth  int
	MaxParamCount int
	Exclude       string
	Rule          string
	Out           io.Writer
	Err           io.Writer
}

// RunCheck runs the custom linter in standalone mode.
// Returns the number of diagnostics and fixed files.
func RunCheck(opts CheckOpts) (diags, fixed int, err error) {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.Err == nil {
		opts.Err = os.Stderr
	}

	cfg, err := BuildLinterConfig(
		opts.Exclude, opts.Rule,
		opts.MaxLineWidth, opts.TabWidth,
		opts.MaxFuncLen, opts.MaxNestDepth, opts.MaxParamCount,
	)
	if err != nil {
		return 0, 0, err
	}
	cfg.Out = opts.Out
	cfg.Err = opts.Err

	diags, fixed, runErr := linter.Run(opts.Path, opts.DoFix, cfg)
	if runErr != nil {
		return 0, 0, runErr
	}

	return diags, fixed, nil
}

// BuildLinterConfig creates a linter.Config from the
// common flag values shared by check and lint commands.
func BuildLinterConfig(
	exclude, rule string,
	maxLineWidth, tabWidth,
	maxFuncLen, maxNestDepth, maxParamCount int,
) (linter.Config, error) {
	excl := ParseExclusions(exclude)
	if err := linter.ValidateExclusions(excl); err != nil {
		return linter.Config{}, err
	}

	if rule != "" {
		single := BuildSingleRuleExclusions(rule)
		if single == nil {
			return linter.Config{},
				fmt.Errorf("unknown rule: %q", rule)
		}
		excl = single
	}

	return linter.Config{
		MaxLineWidth:  maxLineWidth,
		TabWidth:      tabWidth,
		MaxFuncLen:    maxFuncLen,
		MaxNestDepth:  maxNestDepth,
		MaxParamCount: maxParamCount,
		Exclude:       excl,
	}, nil
}

// BuildSingleRuleExclusions returns exclusions for every
// rule except the given one. Returns nil if the rule is
// unknown.
func BuildSingleRuleExclusions(name string) map[linter.Rule]bool {
	r := linter.Rule(name)
	check := map[linter.Rule]bool{r: true}
	if err := linter.ValidateExclusions(check); err != nil {
		return nil
	}
	excl := make(map[linter.Rule]bool)
	for _, ar := range linter.AllRules {
		if ar != r {
			excl[ar] = true
		}
	}
	return excl
}

// ParseExclusions parses a comma-separated list of rule
// names into a map.
func ParseExclusions(csv string) map[linter.Rule]bool {
	excl := make(map[linter.Rule]bool)
	if csv == "" {
		return excl
	}
	for _, r := range strings.Split(csv, ",") {
		excl[linter.Rule(strings.TrimSpace(r))] = true
	}
	return excl
}

// SummarizeOutcome prints a summary and returns the exit
// code.
func SummarizeOutcome(w io.Writer, doFix bool, diags, fixed int) int {
	if doFix && fixed > 0 {
		_, _ = fmt.Fprintf(w, "\n%d file(s) fixed.\n", fixed)
	}
	if diags > 0 {
		_, _ = fmt.Fprintf(w, "\n%d violation(s) found.\n", diags)
		return 1
	}
	return 0
}
