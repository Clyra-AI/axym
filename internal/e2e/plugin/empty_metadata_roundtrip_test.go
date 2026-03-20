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

func TestRelationshipEnvelopeRoundTripsToStoredChain(t *testing.T) {
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
	pluginSource := []byte("package main\nimport \"fmt\"\nfunc main(){fmt.Println(`{\"source_type\":\"plugin\",\"source\":\"custom\",\"source_product\":\"axym\",\"record_type\":\"tool_invocation\",\"agent_id\":\"agent-1\",\"timestamp\":\"2026-03-18T00:00:00Z\",\"event\":{\"tool_name\":\"scan\"},\"metadata\":{},\"relationship\":{\"parent_ref\":{\"kind\":\"trace\",\"id\":\"trace-123\"},\"entity_refs\":[{\"kind\":\"resource\",\"id\":\"db://prod\"}],\"policy_ref\":{\"policy_digest\":\"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\"}},\"controls\":{\"permissions_enforced\":true}}`)}\n")
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
	if data["appended"] != float64(1) {
		t.Fatalf("expected appended=1 output=%s", string(collectOut))
	}

	raw, err := os.ReadFile(filepath.Join(storeDir, "chain.json"))
	if err != nil {
		t.Fatalf("read chain: %v", err)
	}
	var chain struct {
		Records []struct {
			Relationship struct {
				ParentRef *struct {
					Kind string `json:"kind"`
					ID   string `json:"id"`
				} `json:"parent_ref,omitempty"`
				EntityRefs []struct {
					Kind string `json:"kind"`
					ID   string `json:"id"`
				} `json:"entity_refs,omitempty"`
				PolicyRef *struct {
					PolicyDigest string `json:"policy_digest"`
				} `json:"policy_ref,omitempty"`
			} `json:"relationship"`
		} `json:"records"`
	}
	if err := json.Unmarshal(raw, &chain); err != nil {
		t.Fatalf("decode chain: %v", err)
	}
	if len(chain.Records) != 1 {
		t.Fatalf("expected one record, got %d", len(chain.Records))
	}
	relationship := chain.Records[0].Relationship
	if relationship.ParentRef == nil || relationship.ParentRef.Kind != "trace" || relationship.ParentRef.ID != "trace-123" {
		t.Fatalf("parent ref mismatch: %+v", relationship.ParentRef)
	}
	foundResource := false
	for _, ref := range relationship.EntityRefs {
		if ref.Kind == "resource" && ref.ID == "db://prod" {
			foundResource = true
			break
		}
	}
	if !foundResource {
		t.Fatalf("expected resource ref: %+v", relationship.EntityRefs)
	}
	if relationship.PolicyRef == nil || relationship.PolicyRef.PolicyDigest != "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("policy ref mismatch: %+v", relationship.PolicyRef)
	}
}
