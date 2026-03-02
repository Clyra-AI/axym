package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestMapUsesDefaultFrameworksWhenFlagMissing(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"map", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s", exit, exitSuccess, stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	frameworks, _ := data["frameworks"].([]any)
	if len(frameworks) != 2 {
		t.Fatalf("expected default frameworks output: %s", stdout.String())
	}
}

func TestMapJSONOutputDeterministicShape(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	mustCollectFixtures(t, storeDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"map", "--frameworks", "eu-ai-act,soc2", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload["command"] != "map" {
		t.Fatalf("command mismatch: %s", stdout.String())
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok=true: %s", stdout.String())
	}
	data, _ := payload["data"].(map[string]any)
	if _, ok := data["frameworks"].([]any); !ok {
		t.Fatalf("missing frameworks in output: %s", stdout.String())
	}
}

func TestMapThresholdFailureExitAndReason(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	mustCollectFixtures(t, storeDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"map", "--frameworks", "eu-ai-act", "--store-dir", storeDir, "--min-coverage", "1", "--json"}, &stdout, &stderr)
	if exit != exitRegressionDrift {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s", exit, exitRegressionDrift, stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["reason"] != "COVERAGE_THRESHOLD_NOT_MET" {
		t.Fatalf("reason mismatch: %s", stdout.String())
	}
	data, _ := payload["data"].(map[string]any)
	thresholdObj, _ := data["threshold"].(map[string]any)
	if thresholdObj["passed"] != false {
		t.Fatalf("threshold status mismatch: %s", stdout.String())
	}
}

func mustCollectFixtures(t *testing.T, storeDir string) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"collect", "--fixture-dir", fixtureDir(t), "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("collect setup failed: exit=%d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
}
