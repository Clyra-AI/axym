package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Clyra-AI/proof"
)

func TestIngestRequiresSource(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"ingest", "--json"}, &stdout, &stderr)
	if exit != exitInvalidInput {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s", exit, exitInvalidInput, stdout.String())
	}
}

func TestIngestWrkrNoInputContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"ingest", "--source", "wrkr", "--store-dir", storeDir, "--state-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	result, _ := data["result"].(map[string]any)
	reasons, _ := result["reason_codes"].([]any)
	if len(reasons) != 1 || reasons[0] != "NO_INPUT" {
		t.Fatalf("reason mismatch: %s", stdout.String())
	}
}

func TestIngestWrkrAppendsRecords(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	inputPath := filepath.Join(root, "wrkr.jsonl")
	writeWrkrInput(t, inputPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{
		"ingest",
		"--source", "wrkr",
		"--input", inputPath,
		"--store-dir", storeDir,
		"--state-dir", root,
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
	result, _ := data["result"].(map[string]any)
	if appended, _ := result["appended"].(float64); appended != 1 {
		t.Fatalf("appended mismatch: %s", stdout.String())
	}
}

func TestIngestWrkrUsesStoreDirForStateWhenStateDirOmitted(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	inputPath := filepath.Join(root, "wrkr.jsonl")
	writeWrkrInput(t, inputPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{
		"ingest",
		"--source", "wrkr",
		"--input", inputPath,
		"--store-dir", storeDir,
		"--json",
	}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}

	if _, err := os.Stat(filepath.Join(storeDir, "wrkr-last-ingest.json")); err != nil {
		t.Fatalf("expected wrkr state in store dir: %v", err)
	}
}

func TestIngestGaitNoInputContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"ingest", "--source", "gait", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	result, _ := data["result"].(map[string]any)
	reasons, _ := result["reason_codes"].([]any)
	if len(reasons) != 1 || reasons[0] != "NO_INPUT" {
		t.Fatalf("reason mismatch: %s", stdout.String())
	}
}

func TestIngestUsesCommandContextCancellation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	inputDir := filepath.Join(root, "gait")
	if err := os.MkdirAll(inputDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inputDir, "native_records.jsonl"), []byte("{\"type\":\"trace\",\"timestamp\":\"2026-02-28T23:05:00Z\",\"agent_id\":\"agent://executor\",\"event\":{\"tool_name\":\"planner\"}}\n"), 0o600); err != nil {
		t.Fatalf("write native records: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := executeContext(ctx, []string{"ingest", "--source", "gait", "--input", inputDir, "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitRuntimeFailure {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s stderr=%s", exit, exitRuntimeFailure, stdout.String(), stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v output=%s", err, stdout.String())
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["reason"] != "GAIT_CONTEXT_CANCELED" {
		t.Fatalf("reason mismatch: got %v output=%s", errObj["reason"], stdout.String())
	}
}

func writeWrkrInput(t *testing.T, path string) {
	t.Helper()
	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     time.Date(2026, 2, 28, 20, 0, 0, 0, time.UTC),
		Source:        "wrkr",
		SourceProduct: "wrkr",
		Type:          "scan_finding",
		AgentID:       "agent-a",
		Event: map[string]any{
			"finding_id":   "finding-1",
			"principal_id": "agent-a",
			"privilege":    "read",
			"approved":     true,
		},
		Metadata: map[string]any{
			"principal_id": "agent-a",
			"scope":        "read",
		},
		Controls: proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("proof.NewRecord: %v", err)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(string(raw))+"\n"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
}
