package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCollectDryRunJSONNoWrites(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"collect", "--dry-run", "--fixture-dir", fixtureDir(t), "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s", exit, stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if ok, _ := payload["ok"].(bool); !ok {
		t.Fatalf("expected ok=true, payload=%s", stdout.String())
	}
	data, _ := payload["data"].(map[string]any)
	if appended, _ := data["appended"].(float64); appended != 0 {
		t.Fatalf("dry-run appended mismatch: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(storeDir, "chain.json")); !os.IsNotExist(err) {
		t.Fatalf("dry-run should not create chain file: err=%v", err)
	}
}

func TestCollectWriteJSONAppendsRecords(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"collect", "--fixture-dir", fixtureDir(t), "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	if appended, _ := data["appended"].(float64); appended != 7 {
		t.Fatalf("expected 7 appended records, payload=%s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(storeDir, "chain.json")); err != nil {
		t.Fatalf("expected chain file: %v", err)
	}
}

func TestCollectWriteJSONWithoutInputsDoesNotSynthesizeEvidence(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"collect", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	if appended, _ := data["appended"].(float64); appended != 0 {
		t.Fatalf("expected no appended records without source inputs, payload=%s", stdout.String())
	}
	if captured, _ := data["captured"].(float64); captured != 0 {
		t.Fatalf("expected no captured records without source inputs, payload=%s", stdout.String())
	}
}

func TestCollectGovernanceContextEngineeringJSONAppendsRecords(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{
		"collect",
		"--governance-event-file", governanceFixturePath(t),
		"--store-dir", storeDir,
		"--json",
	}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	if appended, _ := data["appended"].(float64); appended != 3 {
		t.Fatalf("expected 3 appended governance records, payload=%s", stdout.String())
	}
}

func TestCollectPluginEmptyMetadataRoundTripsVerify(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	pluginPath := filepath.Join(root, "plugin.go")
	pluginSource := []byte("package main\nimport \"fmt\"\nfunc main(){fmt.Println(`{\"source_type\":\"plugin\",\"source\":\"custom\",\"source_product\":\"axym\",\"record_type\":\"tool_invocation\",\"agent_id\":\"agent-1\",\"timestamp\":\"2026-03-18T00:00:00Z\",\"event\":{\"tool_name\":\"scan\"},\"metadata\":{},\"controls\":{\"permissions_enforced\":true}}`)}\n")
	if err := os.WriteFile(pluginPath, pluginSource, 0o600); err != nil {
		t.Fatalf("write plugin source: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{
		"collect",
		"--store-dir", storeDir,
		"--plugin-timeout", "60s",
		"--plugin", "go run " + pluginPath,
		"--json",
	}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("collect exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}

	var collectPayload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &collectPayload); err != nil {
		t.Fatalf("decode collect json: %v", err)
	}
	data, _ := collectPayload["data"].(map[string]any)
	if appended, _ := data["appended"].(float64); appended != 1 {
		t.Fatalf("expected one appended plugin record, output=%s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	exit = execute([]string{"verify", "--chain", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("verify exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}

	var verifyPayload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &verifyPayload); err != nil {
		t.Fatalf("decode verify json: %v", err)
	}
	verifyData, _ := verifyPayload["data"].(map[string]any)
	verification, _ := verifyData["verification"].(map[string]any)
	if intact, _ := verification["intact"].(bool); !intact {
		t.Fatalf("expected intact chain, output=%s", stdout.String())
	}
	if count, _ := verification["count"].(float64); count != 1 {
		t.Fatalf("expected verified record count=1, output=%s", stdout.String())
	}
}

func fixtureDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "fixtures", "collectors"))
}

func governanceFixturePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "fixtures", "governance", "context_engineering.jsonl"))
}
