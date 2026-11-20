package gocover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mirurobotics/gotools/internal/testutil"
)

func TestGoModule(t *testing.T) {
	makeGoProject(t)

	mod, err := GoModule()
	if err != nil {
		t.Fatalf("GoModule: %v", err)
	}
	if mod != "testmod" {
		t.Errorf("got %q, want %q", mod, "testmod")
	}
}

func TestGoListPackages(t *testing.T) {
	makeGoProject(t)

	pkgs, err := GoListPackages("testmod/...")
	if err != nil {
		t.Fatalf("GoListPackages: %v", err)
	}
	found := false
	for _, pkg := range pkgs {
		if pkg == "testmod/mypkg" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected testmod/mypkg in %v", pkgs)
	}
}

func TestGoListTestPackages(t *testing.T) {
	makeGoProject(t)

	notest := filepath.Join(".", "notest")
	//nolint:gosec // G301: test directory
	_ = os.MkdirAll(notest, 0o755)
	lib := "package notest\n\nfunc Noop() {}\n"
	//nolint:gosec // G306: test file
	_ = os.WriteFile(filepath.Join(notest, "lib.go"), []byte(lib), 0o644)

	pkgs, err := GoListTestPackages("testmod/...")
	if err != nil {
		t.Fatalf("GoListTestPackages: %v", err)
	}
	if len(pkgs) != 1 || pkgs[0] != "testmod/mypkg" {
		t.Errorf("expected only testmod/mypkg, got %v", pkgs)
	}
}

func TestGoListTestPackages_Error(t *testing.T) {
	makeGoProject(t)

	_, err := GoListTestPackages("./does-not-exist/...")
	if err == nil {
		t.Fatal("expected error for a pattern matching no packages")
	}
}

func TestRunTests_Pass(t *testing.T) {
	makeGoProject(t)

	out, err := RunTests([]string{"testmod/mypkg"})
	if err != nil {
		t.Fatalf("RunTests: %v\n%s", err, out)
	}
}

func TestRunTests_Failure(t *testing.T) {
	makeGoProject(t)

	failing := "package mypkg\n\n" +
		"import \"testing\"\n\n" +
		"func TestBroken(t *testing.T) { t.Fatal(\"boom\") }\n"
	//nolint:gosec // G306: test file
	err := os.WriteFile(
		filepath.Join("mypkg", "broken_test.go"), []byte(failing), 0o644,
	)
	if err != nil {
		t.Fatal(err)
	}

	out, err := RunTests([]string{"testmod/mypkg"})
	if err == nil {
		t.Fatal("expected error from failing test")
	}
	if !strings.Contains(string(out), "boom") {
		t.Errorf("output missing failure detail: %s", out)
	}
}

func makeGoProject(t *testing.T) {
	t.Helper()
	lib := "package mypkg\n\nfunc Add(a, b int) int { return a + b }\n"
	testSrc := "package mypkg\n\n" +
		"import \"testing\"\n\n" +
		"func TestAdd(t *testing.T) {\n" +
		"\tif Add(1,2) != 3 { t.Fatal() }\n}\n"
	testutil.WriteTestModule(t, lib, testSrc)
}
