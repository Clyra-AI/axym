package cli

import (
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRootHelpListsPrimaryCommands(t *testing.T) {
	t.Parallel()

	stdout, exit := runAxymCLI(t, "--help")
	if exit != 0 {
		t.Fatalf("unexpected exit %d output=%s", exit, stdout)
	}
	for _, want := range []string{
		"init",
		"collect",
		"record",
		"ingest",
		"map",
		"gaps",
		"regress",
		"review",
		"override",
		"replay",
		"bundle",
		"verify",
		"version",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("missing command %q from help output=%s", want, stdout)
		}
	}
}

func runAxymCLI(t *testing.T, args ...string) (string, int) {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	binaryName := "axym"
	if runtime.GOOS == "windows" {
		binaryName = "axym.exe"
	}
	binaryPath := filepath.Join(t.TempDir(), binaryName)

	build := exec.Command("go", "build", "-o", binaryPath, "./cmd/axym")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build axym: %v output=%s", err, string(out))
	}

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return string(out), exitErr.ExitCode()
	}
	t.Fatalf("run axym: %v", err)
	return "", 1
}
