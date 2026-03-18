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
