package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestGapsUsesDefaultFrameworksWhenFlagMissing(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"gaps", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
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

func TestGapsJSONOutputAndExplain(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	mustCollectFixtures(t, storeDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"gaps", "--frameworks", "eu-ai-act,soc2", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload["command"] != "gaps" {
		t.Fatalf("command mismatch: %s", stdout.String())
	}
	data, _ := payload["data"].(map[string]any)
	if _, ok := data["gaps"].([]any); !ok {
		t.Fatalf("missing gaps list: %s", stdout.String())
	}
	if _, ok := data["grade"].(map[string]any); !ok {
		t.Fatalf("missing grade object: %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	exit = execute([]string{"gaps", "--frameworks", "eu-ai-act", "--store-dir", storeDir, "--explain"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("explain exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), "grade=") {
		t.Fatalf("expected explain output, got %s", stdout.String())
	}
}

func TestGapsThresholdFailureExitAndReason(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	mustCollectFixtures(t, storeDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"gaps", "--frameworks", "soc2", "--store-dir", storeDir, "--min-coverage", "1", "--json"}, &stdout, &stderr)
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
}

func TestGapsRejectsInvalidMinCoverageInput(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	mustCollectFixtures(t, storeDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"gaps", "--store-dir", storeDir, "--min-coverage", "-0.1", "--json"}, &stdout, &stderr)
	if exit != exitInvalidInput {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s stderr=%s", exit, exitInvalidInput, stdout.String(), stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["reason"] != "invalid_input" {
		t.Fatalf("reason mismatch: %s", stdout.String())
	}
}
