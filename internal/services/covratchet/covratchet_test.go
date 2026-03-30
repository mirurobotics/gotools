package covratchet

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadCovgate(t *testing.T) {
	tests := []struct {
		name    string
		content string
		exists  bool
		want    string
	}{
		{"valid value", "85.5\n", true, "85.5"},
		{"zero", "0\n", true, "0"},
		{"missing file", "", false, ""},
		{"empty file", "", true, ""},
		{"with trailing space", "90.0  \n", true, "90.0"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, ".covgate")
			if tc.exists {
				if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got := readCovgate(path)
			if got != tc.want {
				t.Errorf("readCovgate() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestWriteCovgate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".covgate")

	if err := writeCovgate(path, 85.5); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != "85.5\n" {
		t.Errorf("writeCovgate wrote %q, want %q", got, "85.5\n")
	}
}

func TestWriteCovgate_Error(t *testing.T) {
	// Write to a non-existent directory.
	err := writeCovgate("/nonexistent/dir/.covgate", 50.0)
	if err == nil {
		t.Error("expected error writing to non-existent dir")
	}
}
