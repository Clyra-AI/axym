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

func TestEmptyMetadataRoundTripsVerify(t *testing.T) {
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

	pluginPath := filepath.Join(t.TempDir(), "plugin.go")
	pluginSource := []byte("package main\nimport \"fmt\"\nfunc main(){fmt.Println(`{\"source_type\":\"plugin\",\"source\":\"custom\",\"source_product\":\"axym\",\"record_type\":\"tool_invocation\",\"agent_id\":\"agent-1\",\"timestamp\":\"2026-03-18T00:00:00Z\",\"event\":{\"tool_name\":\"scan\"},\"metadata\":{},\"controls\":{\"permissions_enforced\":true}}`)}\n")
	if err := os.WriteFile(pluginPath, pluginSource, 0o600); err != nil {
		t.Fatalf("write plugin source: %v", err)
	}

	storeDir := filepath.Join(t.TempDir(), "store")
	collect := exec.Command(binaryPath,
		"collect",
		"--json",
		"--store-dir", storeDir,
		"--plugin-timeout", "60s",
		"--plugin", "go run "+pluginPath,
	)
	collect.Dir = repoRoot
	collectOut, err := collect.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("collect exit=%d output=%s", exitErr.ExitCode(), string(collectOut))
		}
		t.Fatalf("run collect: %v output=%s", err, string(collectOut))
	}
	var collectPayload map[string]any
	if err := json.Unmarshal(collectOut, &collectPayload); err != nil {
		t.Fatalf("decode collect json: %v output=%s", err, string(collectOut))
	}
	data, _ := collectPayload["data"].(map[string]any)
	if appended, _ := data["appended"].(float64); appended != 1 {
		t.Fatalf("expected appended=1 output=%s", string(collectOut))
	}

	verify := exec.Command(binaryPath, "verify", "--chain", "--store-dir", storeDir, "--json")
	verify.Dir = repoRoot
	verifyOut, err := verify.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Fatalf("verify exit=%d output=%s", exitErr.ExitCode(), string(verifyOut))
		}
		t.Fatalf("run verify: %v output=%s", err, string(verifyOut))
	}
	var verifyPayload map[string]any
	if err := json.Unmarshal(verifyOut, &verifyPayload); err != nil {
		t.Fatalf("decode verify json: %v output=%s", err, string(verifyOut))
	}
	verifyData, _ := verifyPayload["data"].(map[string]any)
	verification, _ := verifyData["verification"].(map[string]any)
	if verification["intact"] != true {
		t.Fatalf("expected intact=true output=%s", string(verifyOut))
	}
	if verification["count"] != float64(1) {
		t.Fatalf("expected count=1 output=%s", string(verifyOut))
	}
}
