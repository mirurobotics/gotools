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

func assertFlagTypes(t *testing.T, fl *pflag.FlagSet, want map[string]string) {
	t.Helper()
	for name, wantType := range want {
		f := fl.Lookup(name)
		if f == nil {
			t.Errorf("flag %q not registered", name)
			continue
		}
		if f.Value.Type() != wantType {
			t.Errorf("flag %q type = %q, want %q", name, f.Value.Type(), wantType)
		}
	}
}

func assertStringDefaults(t *testing.T, fl *pflag.FlagSet, want map[string]string) {
	t.Helper()
	for name, w := range want {
		got, err := fl.GetString(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
		} else if got != w {
			t.Errorf("flag %q default = %q, want %q", name, got, w)
		}
	}
}

func assertBoolDefaults(t *testing.T, fl *pflag.FlagSet, want map[string]bool) {
	t.Helper()
	for name, w := range want {
		got, err := fl.GetBool(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
		} else if got != w {
			t.Errorf("flag %q default = %v, want %v", name, got, w)
		}
	}
}

func assertIntDefaults(t *testing.T, fl *pflag.FlagSet, want map[string]int) {
	t.Helper()
	for name, w := range want {
		got, err := fl.GetInt(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
		} else if got != w {
			t.Errorf("flag %q default = %d, want %d", name, got, w)
		}
	}
}

func assertFloatDefaults(t *testing.T, fl *pflag.FlagSet, want map[string]float64) {
	t.Helper()
	for name, w := range want {
		got, err := fl.GetFloat64(name)
		if err != nil {
			t.Errorf("flag %q: %v", name, err)
		} else if got != w {
			t.Errorf("flag %q default = %v, want %v", name, got, w)
		}
	}
}
