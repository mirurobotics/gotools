package commands

import (
	"testing"

	"github.com/spf13/pflag"
)

func linterIntDefaults() map[string]int {
	return map[string]int{
		"max-line-width":     88,
		"tab-width":          4,
		"max-func-len":       30,
		"max-file-len":       300,
		"max-nest-depth":     2,
		"max-param-count":    6,
		"max-doc-lines":      1,
		"max-doc-file-lines": 20,
	}
}

func assertFlagTypes(t *testing.T, flags *pflag.FlagSet, want map[string]string) {
	t.Helper()
	for name, wantType := range want {
		flag := flags.Lookup(name)
		if flag == nil {
			t.Errorf("flag %q not registered", name)
			continue
		}
		if flag.Value.Type() != wantType {
			t.Errorf("flag %q type = %q, want %q", name, flag.Value.Type(), wantType)
		}
	}
}

func assertStringDefaults(t *testing.T, flags *pflag.FlagSet, want map[string]string) {
	t.Helper()
	for name, wantVal := range want {
		got, err := flags.GetString(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
		} else if got != wantVal {
			t.Errorf("flag %q default = %q, want %q", name, got, wantVal)
		}
	}
}

func assertBoolDefaults(t *testing.T, flags *pflag.FlagSet, want map[string]bool) {
	t.Helper()
	for name, wantVal := range want {
		got, err := flags.GetBool(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
		} else if got != wantVal {
			t.Errorf("flag %q default = %v, want %v", name, got, wantVal)
		}
	}
}

func assertIntDefaults(t *testing.T, flags *pflag.FlagSet, want map[string]int) {
	t.Helper()
	for name, wantVal := range want {
		got, err := flags.GetInt(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
		} else if got != wantVal {
			t.Errorf("flag %q default = %d, want %d", name, got, wantVal)
		}
	}
}

func assertFloatDefaults(t *testing.T, flags *pflag.FlagSet, want map[string]float64) {
	t.Helper()
	for name, wantVal := range want {
		got, err := flags.GetFloat64(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
		} else if got != wantVal {
			t.Errorf("flag %q default = %v, want %v", name, got, wantVal)
		}
	}
}
