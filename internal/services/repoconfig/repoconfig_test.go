package repoconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	//nolint:exhaustruct // test uses partial initialization
	tests := []struct {
		name    string
		setup   func(t *testing.T, dir string)
		want    Config
		wantErr bool
	}{
		{
			name:  "missing file",
			setup: func(*testing.T, string) {},
			want:  Config{},
		},
		{
			name: "sections keyed by command path",
			setup: writeConfig(
				"lint:\n  fix: false\n  max-line-width: 100\n" +
					"deps update:\n  paths: ./cmd\n",
			),
			want: Config{
				"lint":        {"fix": "false", "max-line-width": "100"},
				"deps update": {"paths": "./cmd"},
			},
		},
		{name: "empty file", setup: writeConfig(""), want: Config{}},
		{name: "null value", setup: writeConfig("lint:\n  paths:\n"), wantErr: true},
		{name: "malformed yaml", setup: writeConfig("lint: [\n"), wantErr: true},
		{
			name:    "list value",
			setup:   writeConfig("lint:\n  paths: [a, b]\n"),
			wantErr: true,
		},
		{
			name:    "map value",
			setup:   writeConfig("lint:\n  paths:\n    a: b\n"),
			wantErr: true,
		},
		{
			name: "unreadable file",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				//nolint:gosec // G301: test directory
				if err := os.Mkdir(filepath.Join(dir, FileName), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			tc.setup(t, dir)

			got, err := Load(dir)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Load() = %v, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() returned error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Load() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func writeConfig(content string) func(t *testing.T, dir string) {
	return func(t *testing.T, dir string) {
		t.Helper()
		path := filepath.Join(dir, FileName)
		//nolint:gosec // G306: test file
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
