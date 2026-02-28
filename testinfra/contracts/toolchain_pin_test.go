package contracts

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func testRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(testRepoRoot(t), filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

func TestToolVersionsPinned(t *testing.T) {
	t.Parallel()

	content := readRepoFile(t, ".tool-versions")
	expected := []string{
		"golang 1.25.7",
		"python 3.13.0",
		"nodejs 22.0.0",
	}
	for _, line := range expected {
		if !strings.Contains(content, line) {
			t.Fatalf(".tool-versions missing pinned line: %q", line)
		}
	}
}

func TestNoFloatingDependenciesInGoMod(t *testing.T) {
	t.Parallel()

	goMod := readRepoFile(t, "go.mod")
	if strings.Contains(goMod, "@latest") {
		t.Fatal("go.mod contains floating @latest dependency")
	}
}

func TestProofDependencyPolicy(t *testing.T) {
	t.Parallel()

	goMod := readRepoFile(t, "go.mod")
	re := regexp.MustCompile(`github.com/Clyra-AI/proof\s+v([0-9]+)\.([0-9]+)\.([0-9]+)`)
	match := re.FindStringSubmatch(goMod)
	if len(match) != 4 {
		t.Fatal("github.com/Clyra-AI/proof dependency is missing or unpinned")
	}

	major, err := strconv.Atoi(match[1])
	if err != nil {
		t.Fatalf("parse proof major version: %v", err)
	}
	minor, err := strconv.Atoi(match[2])
	if err != nil {
		t.Fatalf("parse proof minor version: %v", err)
	}
	patch, err := strconv.Atoi(match[3])
	if err != nil {
		t.Fatalf("parse proof patch version: %v", err)
	}
	if major != 0 {
		t.Fatalf("unexpected proof major version: %d", major)
	}
	if minor < 4 || (minor == 4 && patch < 5) {
		t.Fatalf("proof version below policy floor (v0.4.5): v%d.%d.%d", major, minor, patch)
	}
}
