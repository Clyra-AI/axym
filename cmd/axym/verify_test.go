package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Clyra-AI/proof"
)

func TestVerifyChainJSONSuccessWithMissingChain(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exit := execute([]string{"verify", "--chain", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d want %d stderr=%s", exit, exitSuccess, stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if ok, _ := payload["ok"].(bool); !ok {
		t.Fatalf("expected ok=true, payload=%s", stdout.String())
	}
}

func TestVerifyChainTamperExitCodeAndReason(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	chain := proof.NewChain("test")
	r1, err := proof.NewRecord(proof.RecordOpts{
		Source:        "axym",
		SourceProduct: "axym",
		Type:          "tool_invocation",
		Timestamp:     time.Date(2026, 2, 28, 15, 0, 0, 0, time.UTC),
		Event:         map[string]any{"tool_name": "fetch"},
		Controls:      proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("NewRecord r1: %v", err)
	}
	if err := proof.AppendToChain(chain, r1); err != nil {
		t.Fatalf("AppendToChain r1: %v", err)
	}
	r2, err := proof.NewRecord(proof.RecordOpts{
		Source:        "axym",
		SourceProduct: "axym",
		Type:          "tool_invocation",
		Timestamp:     time.Date(2026, 2, 28, 15, 1, 0, 0, time.UTC),
		Event:         map[string]any{"tool_name": "fetch2"},
		Controls:      proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("NewRecord r2: %v", err)
	}
	if err := proof.AppendToChain(chain, r2); err != nil {
		t.Fatalf("AppendToChain r2: %v", err)
	}
	chain.Records[1].Event["tool_name"] = "tampered"
	raw, err := json.MarshalIndent(chain, "", "  ")
	if err != nil {
		t.Fatalf("marshal chain: %v", err)
	}
	if err := os.WriteFile(filepath.Join(storeDir, "chain.json"), raw, 0o600); err != nil {
		t.Fatalf("write chain: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"verify", "--chain", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitVerificationFailed {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s stderr=%s", exit, exitVerificationFailed, stdout.String(), stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	errObj, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error object: %s", stdout.String())
	}
	if errObj["reason"] != "chain_tamper_detected" {
		t.Fatalf("reason mismatch: got %v", errObj["reason"])
	}
}

func TestVerifyBundleIgnoresTempDirWithoutMutation(t *testing.T) {
	t.Parallel()

	unsafeDir := filepath.Join(t.TempDir(), "unsafe")
	if err := os.MkdirAll(unsafeDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(unsafeDir, "foreign.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write foreign file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"verify", "--bundle", bundleFixturePath(t), "--temp-dir", unsafeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s stderr=%s", exit, exitSuccess, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(unsafeDir, ".axym-managed")); !os.IsNotExist(err) {
		t.Fatalf("expected no managed marker creation, got err=%v", err)
	}
}

func TestVerifyBundleFreshStoreDirDoesNotCreateManagedArtifacts(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"verify", "--bundle", bundleFixturePath(t), "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s stderr=%s", exit, exitSuccess, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(storeDir); !os.IsNotExist(err) {
		t.Fatalf("expected verify --bundle to leave store dir untouched, got err=%v", err)
	}
}

func bundleFixturePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "fixtures", "bundles", "good"))
}

func TestVerifyInvalidTargetJSONContract(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"verify", "--json"}, &stdout, &stderr)
	if exit != exitInvalidInput {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s stderr=%s", exit, exitInvalidInput, stdout.String(), stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v output=%s", err, stdout.String())
	}
	if ok, _ := payload["ok"].(bool); ok {
		t.Fatalf("expected ok=false output=%s", stdout.String())
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["reason"] != "invalid_input" {
		t.Fatalf("reason mismatch: got %v output=%s", errObj["reason"], stdout.String())
	}
}

func TestVerifyBundleIncludesComplianceEnvelope(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	bundleDir := filepath.Join(t.TempDir(), "bundle")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"collect", "--fixture-dir", filepath.Join("fixtures", "collectors"), "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("collect setup failed: exit=%d stdout=%s stderr=%s", exit, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	exit = execute([]string{"bundle", "--audit", "Q3-2026", "--frameworks", "eu-ai-act,soc2", "--store-dir", storeDir, "--output", bundleDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("bundle setup failed: exit=%d stdout=%s stderr=%s", exit, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	exit = execute([]string{"verify", "--bundle", bundleDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("verify failed: exit=%d stdout=%s stderr=%s", exit, stdout.String(), stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode verify output: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	verification, _ := data["verification"].(map[string]any)
	if verification["compliance_verified"] != true {
		t.Fatalf("expected compliance_verified=true output=%s", stdout.String())
	}
	if verification["oscal_valid"] != true {
		t.Fatalf("expected oscal_valid=true output=%s", stdout.String())
	}
}

func TestVerifyBundleUsesBundleDeclaredFrameworksWhenFlagOmitted(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	bundleDir := filepath.Join(t.TempDir(), "bundle")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"collect", "--fixture-dir", filepath.Join("fixtures", "collectors"), "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("collect setup failed: exit=%d stdout=%s stderr=%s", exit, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	exit = execute([]string{"bundle", "--audit", "Q3-2026", "--frameworks", "sox,pci-dss", "--store-dir", storeDir, "--output", bundleDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("bundle setup failed: exit=%d stdout=%s stderr=%s", exit, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	exit = execute([]string{"verify", "--bundle", bundleDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("verify should succeed with bundle-declared frameworks: exit=%d stdout=%s stderr=%s", exit, stdout.String(), stderr.String())
	}
}
