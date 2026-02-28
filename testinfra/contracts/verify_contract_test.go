package contracts

import (
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestVerifyJSONEnvelopeContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	stdout, exit := runAxymContract(t, "verify", "--chain", "--store-dir", storeDir, "--json")
	if exit != 0 {
		t.Fatalf("unexpected exit %d output=%s", exit, stdout)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode output json: %v", err)
	}
	if payload["command"] != "verify" {
		t.Fatalf("command mismatch: got %v", payload["command"])
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok=true, got %v", payload["ok"])
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data envelope: %s", stdout)
	}
	if data["target"] != "chain" {
		t.Fatalf("target mismatch: got %v", data["target"])
	}
}

func TestVerifyInvalidTargetContractExitCode(t *testing.T) {
	t.Parallel()

	stdout, exit := runAxymContract(t, "verify", "--json")
	if exit != 6 {
		t.Fatalf("exit mismatch: got %d want 6 output=%s", exit, stdout)
	}
}

func runAxymContract(t *testing.T, args ...string) (string, int) {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	binaryPath := filepath.Join(t.TempDir(), "axym")
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
