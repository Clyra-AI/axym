package plugin

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMalformedJSONLRejected(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	binaryPath := filepath.Join(t.TempDir(), "axym")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}
	build := exec.Command("go", "build", "-o", binaryPath, "./cmd/axym")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build axym: %v output=%s", err, string(out))
	}

	pluginPath := filepath.Join(t.TempDir(), "bad_plugin.go")
	pluginSource := []byte("package main\nimport \"fmt\"\nfunc main(){fmt.Println(\"{bad-json\")}\n")
	if err := os.WriteFile(pluginPath, pluginSource, 0o600); err != nil {
		t.Fatalf("write plugin source: %v", err)
	}

	storeDir := filepath.Join(t.TempDir(), "store")
	cmd := exec.Command(binaryPath,
		"collect",
		"--json",
		"--fixture-dir", filepath.Join(repoRoot, "fixtures", "collectors"),
		"--store-dir", storeDir,
		"--plugin-timeout", "60s",
		"--plugin", "go run "+pluginPath,
	)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("unexpected non-zero exit=%d output=%s", exitErr.ExitCode(), string(out))
		}
		t.Fatalf("run collect: %v output=%s", err, string(out))
	}

	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode collect json: %v output=%s", err, string(out))
	}
	if ok, _ := payload["ok"].(bool); !ok {
		t.Fatalf("expected ok=true output=%s", string(out))
	}
	data, _ := payload["data"].(map[string]any)
	if failures, _ := data["failures"].(float64); failures < 1 {
		t.Fatalf("expected plugin failure surface, output=%s", string(out))
	}
	reasons, _ := data["reason_codes"].([]any)
	found := false
	for _, reason := range reasons {
		if reason == "PLUGIN_MALFORMED_JSONL" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing PLUGIN_MALFORMED_JSONL reason: output=%s", string(out))
	}
	if appended, _ := data["appended"].(float64); appended != 7 {
		t.Fatalf("expected only built-in records appended, output=%s", string(out))
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
