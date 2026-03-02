package regress

import (
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRegressRunExits5OnDrift(t *testing.T) {
	t.Parallel()

	repoRoot := testRepoRoot(t)
	root := t.TempDir()
	storeBaseline := filepath.Join(root, "baseline-store")
	storeCurrent := filepath.Join(root, "current-store")
	baselinePath := filepath.Join(root, "baseline.json")
	frameworkPath := filepath.Join(repoRoot, "fixtures", "frameworks", "regress-minimal.yaml")
	recordPath := filepath.Join(repoRoot, "fixtures", "records", "decision.json")

	if out, exit := runAxym(t, repoRoot, "record", "add", "--input", recordPath, "--store-dir", storeBaseline, "--json"); exit != 0 {
		t.Fatalf("record add exit mismatch: %d output=%s", exit, out)
	}
	if out, exit := runAxym(t, repoRoot, "regress", "init", "--baseline", baselinePath, "--frameworks", frameworkPath, "--store-dir", storeBaseline, "--json"); exit != 0 {
		t.Fatalf("regress init exit mismatch: %d output=%s", exit, out)
	}
	if out, exit := runAxym(t, repoRoot, "regress", "run", "--baseline", baselinePath, "--frameworks", frameworkPath, "--store-dir", storeCurrent, "--json"); exit != 5 {
		t.Fatalf("regress run drift exit mismatch: got %d want 5 output=%s", exit, out)
	}
	out, _ := runAxym(t, repoRoot, "regress", "run", "--baseline", baselinePath, "--frameworks", frameworkPath, "--store-dir", storeCurrent, "--json")
	var payload map[string]any
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("decode output json: %v output=%s", err, out)
	}
	if payload["ok"] != false {
		t.Fatalf("expected ok=false output=%s", out)
	}
}

func testRepoRoot(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
}

func runAxym(t *testing.T, repoRoot string, args ...string) (string, int) {
	t.Helper()
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
