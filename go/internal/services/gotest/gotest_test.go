package gotest

import (
	"reflect"
	"testing"
)

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name  string
		extra []string
		want  []string
	}{
		{
			"strips leading --",
			[]string{"--", "-v", "-count=1"},
			[]string{"test", "./...", "-v", "-count=1"},
		},
		{
			"passes through without --",
			[]string{"-v", "-run=TestFoo"},
			[]string{"test", "./...", "-v", "-run=TestFoo"},
		},
		{"empty extra args", nil, []string{"test", "./..."}},
		{"-- only", []string{"--"}, []string{"test", "./..."}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := buildArgs(tc.extra)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("buildArgs(%v) = %v, want %v", tc.extra, got, tc.want)
			}
		})
	}
}
