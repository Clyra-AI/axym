package review

import (
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEmptyDayContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	stdout, exit := runAxymReview(t, "review", "--date", "2026-09-15", "--store-dir", storeDir, "--json")
	if exit != 0 {
		t.Fatalf("exit mismatch: got=%d output=%s", exit, stdout)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode output json: %v", err)
	}
	if payload["command"] != "review" {
		t.Fatalf("command mismatch: %s", stdout)
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok=true: %s", stdout)
	}
	data, _ := payload["data"].(map[string]any)
	if empty, _ := data["empty"].(bool); !empty {
		t.Fatalf("expected empty day output: %s", stdout)
	}
	if got, _ := data["record_count"].(float64); int(got) != 0 {
		t.Fatalf("record count mismatch: %s", stdout)
	}
	exceptions, _ := data["exceptions"].([]any)
	if len(exceptions) == 0 {
		t.Fatalf("missing exception classes: %s", stdout)
	}
	for _, row := range exceptions {
		item, _ := row.(map[string]any)
		if int(item["count"].(float64)) != 0 {
			t.Fatalf("empty day must keep zero exceptions: %s", stdout)
		}
	}
}

func runAxymReview(t *testing.T, args ...string) (string, int) {
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
