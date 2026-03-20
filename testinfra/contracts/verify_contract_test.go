package contracts

import (
	"encoding/json"
	"errors"
	"os"
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

func TestVerifyChainContractAfterRecordAddWithEmptyMetadata(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	recordPath := filepath.Join(root, "record.json")
	recordPayload := []byte(`{
  "record_id": "contract-record-001",
  "record_version": "v1",
  "timestamp": "2026-03-18T00:00:00Z",
  "source": "manual",
  "source_product": "axym",
  "agent_id": "agent-1",
  "record_type": "approval",
  "event": {"decision": "allow"},
  "metadata": {},
  "controls": {"permissions_enforced": true}
}`)
	if err := os.WriteFile(recordPath, recordPayload, 0o600); err != nil {
		t.Fatalf("write record fixture: %v", err)
	}

	storeDir := filepath.Join(root, "store")
	recordOut, recordExit := runAxymContract(t, "record", "add", "--input", recordPath, "--store-dir", storeDir, "--json")
	if recordExit != 0 {
		t.Fatalf("unexpected record add exit %d output=%s", recordExit, recordOut)
	}
	verifyOut, verifyExit := runAxymContract(t, "verify", "--chain", "--store-dir", storeDir, "--json")
	if verifyExit != 0 {
		t.Fatalf("unexpected verify exit %d output=%s", verifyExit, verifyOut)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(verifyOut), &payload); err != nil {
		t.Fatalf("decode verify json: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	verification, _ := data["verification"].(map[string]any)
	if verification["intact"] != true {
		t.Fatalf("expected intact=true output=%s", verifyOut)
	}
	if verification["count"] != float64(1) {
		t.Fatalf("expected count=1 output=%s", verifyOut)
	}
}

func TestVerifyBundleDoesNotMutateStoreOrTempPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	tempDir := filepath.Join(root, "unsafe-temp")
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		t.Fatalf("mkdir temp dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "foreign.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write temp fixture: %v", err)
	}

	out, exit := runAxymContract(t, "verify", "--bundle", filepath.Join(testRepoRoot(t), "fixtures", "bundles", "good"), "--store-dir", storeDir, "--temp-dir", tempDir, "--json")
	if exit != 0 {
		t.Fatalf("unexpected verify exit %d output=%s", exit, out)
	}
	if _, err := os.Stat(filepath.Join(storeDir, "tmp", "verify", ".axym-managed")); !os.IsNotExist(err) {
		t.Fatalf("expected no managed marker under store dir, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, ".axym-managed")); !os.IsNotExist(err) {
		t.Fatalf("expected no managed marker under temp dir, got err=%v", err)
	}
}

func TestVerifyRejectsSignatureTamperWithVerificationFailureExit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	collectOut, collectExit := runAxymContract(t, "collect", "--fixture-dir", filepath.Join(testRepoRoot(t), "fixtures", "collectors"), "--store-dir", storeDir, "--json")
	if collectExit != 0 {
		t.Fatalf("collect setup exit=%d output=%s", collectExit, collectOut)
	}

	chainPath := filepath.Join(storeDir, "chain.json")
	raw, err := os.ReadFile(chainPath)
	if err != nil {
		t.Fatalf("read chain: %v", err)
	}
	var chain map[string]any
	if err := json.Unmarshal(raw, &chain); err != nil {
		t.Fatalf("decode chain: %v", err)
	}
	records := chain["records"].([]any)
	record := records[0].(map[string]any)
	integrity := record["integrity"].(map[string]any)
	integrity["signature"] = "base64:AAAA"
	tampered, err := json.MarshalIndent(chain, "", "  ")
	if err != nil {
		t.Fatalf("marshal chain: %v", err)
	}
	if err := os.WriteFile(chainPath, tampered, 0o600); err != nil {
		t.Fatalf("write chain: %v", err)
	}

	verifyOut, verifyExit := runAxymContract(t, "verify", "--chain", "--store-dir", storeDir, "--json")
	if verifyExit != 2 {
		t.Fatalf("exit mismatch: got %d want 2 output=%s", verifyExit, verifyOut)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(verifyOut), &payload); err != nil {
		t.Fatalf("decode verify json: %v", err)
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["reason"] != "chain_signature_invalid" {
		t.Fatalf("reason mismatch: got %v output=%s", errObj["reason"], verifyOut)
	}
}

func runAxymContract(t *testing.T, args ...string) (string, int) {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
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
