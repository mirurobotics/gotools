package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/mirurobotics/gotools/internal/services/repoconfig"
	"github.com/mirurobotics/gotools/internal/testutil"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

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

func TestApplyRepoConfig(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	tests := []struct {
		name    string
		config  string
		args    []string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "missing file keeps defaults",
			args: []string{"covgate"},
			want: map[string]string{"parallelism": "0", "packages": "./..."},
		},
		{
			name:   "section sets flag defaults",
			config: "covgate:\n  parallelism: 4\n  packages: ./cmd/...\n",
			args:   []string{"covgate"},
			want:   map[string]string{"parallelism": "4", "packages": "./cmd/..."},
		},
		{
			name:   "command line wins",
			config: "covgate:\n  parallelism: 4\n",
			args:   []string{"covgate", "--parallelism", "2"},
			want:   map[string]string{"parallelism": "2"},
		},
		{
			name:   "other sections are not applied",
			config: "lint:\n  paths: ./cmd\ncovratchet:\n  packages: ./cmd/...\n",
			args:   []string{"covgate"},
			want:   map[string]string{"packages": "./..."},
		},
		{
			name:   "nested command section is known",
			config: "deps update: {}\n",
			args:   []string{"deps", "update"},
			want:   map[string]string{},
		},
		{
			name:    "unknown command",
			config:  "nosuch:\n  paths: ./cmd\n",
			args:    []string{"covgate"},
			wantErr: true,
		},
		{
			name:    "unknown subcommand",
			config:  "deps nosuch: {}\n",
			args:    []string{"covgate"},
			wantErr: true,
		},
		{
			name:    "empty command path",
			config:  "\"\": {}\n",
			args:    []string{"covgate"},
			wantErr: true,
		},
		{
			name:    "unknown flag in running section",
			config:  "covgate:\n  nosuch: x\n",
			args:    []string{"covgate"},
			wantErr: true,
		},
		{
			name:    "unknown flag in other section",
			config:  "lint:\n  nosuch: x\n",
			args:    []string{"covgate"},
			wantErr: true,
		},
		{
			name:    "invalid value",
			config:  "covgate:\n  parallelism: many\n",
			args:    []string{"covgate"},
			wantErr: true,
		},
		{
			name:    "malformed file",
			config:  "covgate: [\n",
			args:    []string{"covgate"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			flags, err := executeWithRepoConfig(t, tc.config, tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatal("Execute() returned nil error, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Execute() returned error: %v", err)
			}
			for name, want := range tc.want {
				if got := flags.Lookup(name).Value.String(); got != want {
					t.Errorf("flag %q = %q, want %q", name, got, want)
				}
			}
		})
	}
}

func executeWithRepoConfig(
	t *testing.T, config string, args []string,
) (*pflag.FlagSet, error) {
	t.Helper()
	dir := testutil.MakePkgDirs(t)
	if config != "" {
		path := filepath.Join(dir, repoconfig.FileName)
		//nolint:gosec // G306: test file
		if err := os.WriteFile(path, []byte(config), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	root := NewRootCommand()
	target, _, err := root.Find(args)
	if err != nil {
		t.Fatal(err)
	}
	target.RunE = func(*cobra.Command, []string) error { return nil }
	root.SetArgs(args)
	return target.Flags(), root.Execute()
}

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

	err := cmd.Help()
	if err != nil {
		t.Fatalf("Help() returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("help output is empty")
	}
}
