package commands

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// ---------------------------------------------------------------------------
// Root command
// ---------------------------------------------------------------------------

func TestNewRootCommand_Metadata(t *testing.T) {
	cmd := NewRootCommand()

	if cmd.Use != "miru" {
		t.Errorf("Use = %q, want %q", cmd.Use, "miru")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
	if cmd.Long == "" {
		t.Error("Long description is empty")
	}
	if !cmd.SilenceErrors {
		t.Error("SilenceErrors should be true")
	}
	if !cmd.SilenceUsage {
		t.Error("SilenceUsage should be true")
	}
}

func TestNewRootCommand_Subcommands(t *testing.T) {
	cmd := NewRootCommand()

	want := map[string]bool{
		"lint":       false,
		"check":      false,
		"covgate":    false,
		"covratchet": false,
		"coverage":   false,
		"test":       false,
		"deps":       false,
		"tag":        false,
	}

	for _, sub := range cmd.Commands() {
		name := sub.Name()
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q not registered on root", name)
		}
	}
}

func TestNewRootCommand_Help(t *testing.T) {
	cmd := NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("--help returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("help output is empty")
	}
}

// ---------------------------------------------------------------------------
// Check command
// ---------------------------------------------------------------------------

func TestNewCheckCommand_Metadata(t *testing.T) {
	cmd := NewCheckCommand()

	if cmd.Use != "check" {
		t.Errorf("Use = %q, want %q", cmd.Use, "check")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
	if cmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

func TestNewCheckCommand_Flags(t *testing.T) {
	cmd := NewCheckCommand()
	fl := cmd.Flags()

	tests := []struct {
		name     string
		wantType string
	}{
		{"path", "string"},
		{"fix", "bool"},
		// Shared linter config flags
		{"max-line-width", "int"},
		{"tab-width", "int"},
		{"max-func-len", "int"},
		{"max-nest-depth", "int"},
		{"max-param-count", "int"},
		{"exclude", "string"},
		{"rule", "string"},
	}

	for _, tc := range tests {
		f := fl.Lookup(tc.name)
		if f == nil {
			t.Errorf("flag %q not registered", tc.name)
			continue
		}
		if f.Value.Type() != tc.wantType {
			t.Errorf("flag %q type = %q, want %q", tc.name, f.Value.Type(), tc.wantType)
		}
	}
}

func TestNewCheckCommand_FlagDefaults(t *testing.T) {
	cmd := NewCheckCommand()
	fl := cmd.Flags()

	stringDefaults := map[string]string{"path": ".", "exclude": "", "rule": ""}
	for name, want := range stringDefaults {
		got, err := fl.GetString(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
			continue
		}
		if got != want {
			t.Errorf("flag %q default = %q, want %q", name, got, want)
		}
	}

	boolDefaults := map[string]bool{"fix": false}
	for name, want := range boolDefaults {
		got, err := fl.GetBool(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
			continue
		}
		if got != want {
			t.Errorf("flag %q default = %v, want %v", name, got, want)
		}
	}

	intDefaults := map[string]int{
		"max-line-width":  88,
		"tab-width":       4,
		"max-func-len":    50,
		"max-nest-depth":  4,
		"max-param-count": 6,
	}
	for name, want := range intDefaults {
		got, err := fl.GetInt(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
			continue
		}
		if got != want {
			t.Errorf("flag %q default = %d, want %d", name, got, want)
		}
	}
}

func TestNewCheckCommand_Help(t *testing.T) {
	cmd := NewCheckCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Help()
	if err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("help output is empty")
	}
}

// ---------------------------------------------------------------------------
// Lint command
// ---------------------------------------------------------------------------

func TestNewLintCommand_Metadata(t *testing.T) {
	cmd := NewLintCommand()

	if cmd.Use != "lint" {
		t.Errorf("Use = %q, want %q", cmd.Use, "lint")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
	if cmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

func TestNewLintCommand_Flags(t *testing.T) {
	cmd := NewLintCommand()
	fl := cmd.Flags()

	tests := []struct {
		name     string
		wantType string
	}{
		{"paths", "string"},
		{"fix", "bool"},
		{"deadcode", "bool"},
		{"deadcode-exclude", "string"},
		{"no-gofumpt", "bool"},
		{"no-golangci", "bool"},
		{"new-from-rev", "string"},
		// Shared linter config flags
		{"max-line-width", "int"},
		{"tab-width", "int"},
		{"max-func-len", "int"},
		{"max-nest-depth", "int"},
		{"max-param-count", "int"},
		{"exclude", "string"},
		{"rule", "string"},
	}

	for _, tc := range tests {
		f := fl.Lookup(tc.name)
		if f == nil {
			t.Errorf("flag %q not registered", tc.name)
			continue
		}
		if f.Value.Type() != tc.wantType {
			t.Errorf("flag %q type = %q, want %q", tc.name, f.Value.Type(), tc.wantType)
		}
	}
}

func TestNewLintCommand_FlagDefaults(t *testing.T) {
	cmd := NewLintCommand()
	fl := cmd.Flags()

	stringDefaults := map[string]string{
		"paths":            "",
		"deadcode-exclude": "",
		"new-from-rev":     "",
		"exclude":          "",
		"rule":             "",
	}
	for name, want := range stringDefaults {
		got, err := fl.GetString(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
			continue
		}
		if got != want {
			t.Errorf("flag %q default = %q, want %q", name, got, want)
		}
	}

	boolDefaults := map[string]bool{
		"fix":         true, // lint defaults to fix=true
		"deadcode":    true,
		"no-gofumpt":  false,
		"no-golangci": false,
	}
	for name, want := range boolDefaults {
		got, err := fl.GetBool(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
			continue
		}
		if got != want {
			t.Errorf("flag %q default = %v, want %v", name, got, want)
		}
	}

	intDefaults := map[string]int{
		"max-line-width":  88,
		"tab-width":       4,
		"max-func-len":    50,
		"max-nest-depth":  4,
		"max-param-count": 6,
	}
	for name, want := range intDefaults {
		got, err := fl.GetInt(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
			continue
		}
		if got != want {
			t.Errorf("flag %q default = %d, want %d", name, got, want)
		}
	}
}

func TestNewLintCommand_Help(t *testing.T) {
	cmd := NewLintCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Help()
	if err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("help output is empty")
	}
}

// ---------------------------------------------------------------------------
// Covgate command
// ---------------------------------------------------------------------------

func TestNewCovgateCommand_Metadata(t *testing.T) {
	cmd := NewCovgateCommand()

	if cmd.Use != "covgate" {
		t.Errorf("Use = %q, want %q", cmd.Use, "covgate")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
	if cmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

func TestNewCovgateCommand_Flags(t *testing.T) {
	cmd := NewCovgateCommand()
	fl := cmd.Flags()

	tests := []struct {
		name     string
		wantType string
	}{
		{"packages", "string"},
		{"exclude", "string"},
		{"src-prefix", "string"},
		{"test-dir", "string"},
		{"default-threshold", "float64"},
		{"parallelism", "int"},
	}

	for _, tc := range tests {
		f := fl.Lookup(tc.name)
		if f == nil {
			t.Errorf("flag %q not registered", tc.name)
			continue
		}
		if f.Value.Type() != tc.wantType {
			t.Errorf("flag %q type = %q, want %q", tc.name, f.Value.Type(), tc.wantType)
		}
	}
}

func TestNewCovgateCommand_FlagDefaults(t *testing.T) {
	cmd := NewCovgateCommand()
	fl := cmd.Flags()

	stringDefaults := map[string]string{
		"packages":   "./...",
		"exclude":    "",
		"src-prefix": "pkg",
		"test-dir":   "",
	}
	for name, want := range stringDefaults {
		got, err := fl.GetString(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
			continue
		}
		if got != want {
			t.Errorf("flag %q default = %q, want %q", name, got, want)
		}
	}

	got, err := fl.GetFloat64("default-threshold")
	if err != nil {
		t.Errorf("flag default-threshold: %v", err)
	} else if got != 80.0 {
		t.Errorf("flag default-threshold default = %f, want %f", got, 80.0)
	}

	gotP, errP := fl.GetInt("parallelism")
	if errP != nil {
		t.Errorf("flag parallelism: %v", errP)
	} else if gotP != 0 {
		t.Errorf("flag parallelism default = %d, want 0", gotP)
	}
}

func TestNewCovgateCommand_Help(t *testing.T) {
	cmd := NewCovgateCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Help()
	if err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("help output is empty")
	}
}

// ---------------------------------------------------------------------------
// Covratchet command
// ---------------------------------------------------------------------------

func TestNewCovratchetCommand_Metadata(t *testing.T) {
	cmd := NewCovratchetCommand()

	if cmd.Use != "covratchet" {
		t.Errorf("Use = %q, want %q", cmd.Use, "covratchet")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
	if cmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

func TestNewCovratchetCommand_Flags(t *testing.T) {
	cmd := NewCovratchetCommand()
	fl := cmd.Flags()

	tests := []struct {
		name     string
		wantType string
	}{
		{"packages", "string"},
		{"src-prefix", "string"},
		{"test-dir", "string"},
		{"parallelism", "int"},
	}

	for _, tc := range tests {
		f := fl.Lookup(tc.name)
		if f == nil {
			t.Errorf("flag %q not registered", tc.name)
			continue
		}
		if f.Value.Type() != tc.wantType {
			t.Errorf("flag %q type = %q, want %q", tc.name, f.Value.Type(), tc.wantType)
		}
	}
}

func TestNewCovratchetCommand_FlagDefaults(t *testing.T) {
	cmd := NewCovratchetCommand()
	fl := cmd.Flags()

	stringDefaults := map[string]string{
		"packages":   "./...",
		"src-prefix": "pkg",
		"test-dir":   "",
	}
	for name, want := range stringDefaults {
		got, err := fl.GetString(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
			continue
		}
		if got != want {
			t.Errorf("flag %q default = %q, want %q", name, got, want)
		}
	}

	gotP, errP := fl.GetInt("parallelism")
	if errP != nil {
		t.Errorf("flag parallelism: %v", errP)
	} else if gotP != 0 {
		t.Errorf("flag parallelism default = %d, want 0", gotP)
	}
}

func TestNewCovratchetCommand_NoDefaultThreshold(t *testing.T) {
	// covratchet does NOT have a default-threshold flag (unlike covgate)
	cmd := NewCovratchetCommand()
	f := cmd.Flags().Lookup("default-threshold")
	if f != nil {
		t.Error("covratchet should not have a default-threshold flag")
	}
}

func TestNewCovratchetCommand_Help(t *testing.T) {
	cmd := NewCovratchetCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Help()
	if err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("help output is empty")
	}
}

// ---------------------------------------------------------------------------
// Coverage command
// ---------------------------------------------------------------------------

func TestNewCoverageCommand_Metadata(t *testing.T) {
	cmd := NewCoverageCommand()

	if cmd.Use != "coverage [sub-package]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "coverage [sub-package]")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
	if cmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

func TestNewCoverageCommand_Flags(t *testing.T) {
	cmd := NewCoverageCommand()
	fl := cmd.Flags()

	tests := []struct {
		name     string
		wantType string
	}{
		{"src-prefix", "string"},
		{"test-prefix", "string"},
		{"no-html", "bool"},
	}

	for _, tc := range tests {
		f := fl.Lookup(tc.name)
		if f == nil {
			t.Errorf("flag %q not registered", tc.name)
			continue
		}
		if f.Value.Type() != tc.wantType {
			t.Errorf("flag %q type = %q, want %q", tc.name, f.Value.Type(), tc.wantType)
		}
	}
}

func TestNewCoverageCommand_FlagDefaults(t *testing.T) {
	cmd := NewCoverageCommand()
	fl := cmd.Flags()

	stringDefaults := map[string]string{"src-prefix": "pkg", "test-prefix": "tests"}
	for name, want := range stringDefaults {
		got, err := fl.GetString(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
			continue
		}
		if got != want {
			t.Errorf("flag %q default = %q, want %q", name, got, want)
		}
	}

	got, err := fl.GetBool("no-html")
	if err != nil {
		t.Errorf("flag no-html: %v", err)
	} else if got != false {
		t.Errorf("flag no-html default = %v, want false", got)
	}
}

func TestNewCoverageCommand_AcceptsMaxOneArg(t *testing.T) {
	cmd := NewCoverageCommand()
	if cmd.Args == nil {
		t.Error("Args validator is nil; expected MaximumNArgs(1)")
	}
}

func TestNewCoverageCommand_Help(t *testing.T) {
	cmd := NewCoverageCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Help()
	if err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("help output is empty")
	}
}

// ---------------------------------------------------------------------------
// Test command
// ---------------------------------------------------------------------------

func TestNewTestCommand_Metadata(t *testing.T) {
	cmd := NewTestCommand()

	if cmd.Use != "test [-- extra args]" {
		t.Errorf("Use = %q, want %q", cmd.Use, "test [-- extra args]")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
	if cmd.RunE == nil {
		t.Error("RunE is nil")
	}
}

func TestNewTestCommand_DisablesFlagParsing(t *testing.T) {
	cmd := NewTestCommand()
	if !cmd.DisableFlagParsing {
		t.Error("DisableFlagParsing should be true")
	}
}

func TestNewTestCommand_Help(t *testing.T) {
	cmd := NewTestCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Because flag parsing is disabled, we call Help() directly
	err := cmd.Help()
	if err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("help output is empty")
	}
}

// ---------------------------------------------------------------------------
// Tag command
// ---------------------------------------------------------------------------

func TestNewTagCommand_Metadata(t *testing.T) {
	cmd := NewTagCommand()

	if cmd.Use != "tag" {
		t.Errorf("Use = %q, want %q", cmd.Use, "tag")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
}

func TestNewTagCommand_Subcommands(t *testing.T) {
	cmd := NewTagCommand()

	want := map[string]bool{"previous": false, "latest": false, "current": false}

	for _, sub := range cmd.Commands() {
		name := sub.Name()
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q not registered on tag", name)
		}
	}
}

func TestNewTagCommand_PreviousAcceptsMaxOneArg(t *testing.T) {
	cmd := NewTagCommand()
	var prev *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "previous" {
			prev = sub
			break
		}
	}
	if prev == nil {
		t.Fatal("previous subcommand not found")
	}
	if prev.Args == nil {
		t.Error("previous Args validator is nil; expected MaximumNArgs(1)")
	}
}

func TestNewTagCommand_LatestAcceptsMaxOneArg(t *testing.T) {
	cmd := NewTagCommand()
	var latest *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "latest" {
			latest = sub
			break
		}
	}
	if latest == nil {
		t.Fatal("latest subcommand not found")
	}
	if latest.Args == nil {
		t.Error("latest Args validator is nil; expected MaximumNArgs(1)")
	}
}

func TestNewTagCommand_Help(t *testing.T) {
	cmd := NewTagCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Help()
	if err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("help output is empty")
	}
}

// ---------------------------------------------------------------------------
// Deps command
// ---------------------------------------------------------------------------

func TestNewDepsCommand_Metadata(t *testing.T) {
	cmd := NewDepsCommand()

	if cmd.Use != "deps" {
		t.Errorf("Use = %q, want %q", cmd.Use, "deps")
	}
	if cmd.Short == "" {
		t.Error("Short description is empty")
	}
}

func TestNewDepsCommand_Subcommands(t *testing.T) {
	cmd := NewDepsCommand()

	want := map[string]bool{"update": false}

	for _, sub := range cmd.Commands() {
		name := sub.Name()
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}

	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q not registered on deps", name)
		}
	}
}

func TestNewDepsCommand_UpdateHasRunE(t *testing.T) {
	cmd := NewDepsCommand()
	var update *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Name() == "update" {
			update = sub
			break
		}
	}
	if update == nil {
		t.Fatal("update subcommand not found")
	}
	if update.RunE == nil {
		t.Error("update RunE is nil")
	}
}

func TestNewDepsCommand_Help(t *testing.T) {
	cmd := NewDepsCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := cmd.Help()
	if err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("help output is empty")
	}
}

// ---------------------------------------------------------------------------
// Shared linter config flags (check + lint both bind them)
// ---------------------------------------------------------------------------

func TestSharedLinterFlags_ConsistentBetweenCheckAndLint(t *testing.T) {
	check := NewCheckCommand()
	lint := NewLintCommand()

	sharedFlags := []string{
		"max-line-width",
		"tab-width",
		"max-func-len",
		"max-nest-depth",
		"max-param-count",
		"exclude",
		"rule",
	}

	for _, name := range sharedFlags {
		cf := check.Flags().Lookup(name)
		lf := lint.Flags().Lookup(name)

		if cf == nil {
			t.Errorf("check command missing shared flag %q", name)
			continue
		}
		if lf == nil {
			t.Errorf("lint command missing shared flag %q", name)
			continue
		}
		if cf.DefValue != lf.DefValue {
			t.Errorf(
				"flag %q default mismatch: check=%q, lint=%q",
				name, cf.DefValue, lf.DefValue,
			)
		}
	}
}
