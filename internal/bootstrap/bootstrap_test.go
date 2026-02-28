package bootstrap

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestRequiredScaffoldPathsExist(t *testing.T) {
	t.Parallel()

	required := []string{
		"cmd/axym",
		"core",
		"internal",
		"schemas/v1",
		"scripts",
		"testinfra",
		"scenarios/axym",
		".github/workflows",
	}

	for _, path := range required {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			info, err := os.Stat(filepath.Join(repoRoot(t), filepath.FromSlash(path)))
			if err != nil {
				t.Fatalf("required path missing: %s: %v", path, err)
			}
			if !info.IsDir() {
				t.Fatalf("required path is not a directory: %s", path)
			}
		})
	}
}

func TestEntrypointHasMainFunction(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(repoRoot(t), filepath.FromSlash("cmd/axym/main.go")), nil, parser.AllErrors)
	if err != nil {
		t.Fatalf("parse cmd/axym/main.go: %v", err)
	}

	if file.Name.Name != "main" {
		t.Fatalf("expected package main, got %q", file.Name.Name)
	}
}
