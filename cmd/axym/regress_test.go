package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRegressBaselineRequired(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"regress", "run", "--json"}, &stdout, &stderr)
	if exit != exitInvalidInput {
		t.Fatalf("exit mismatch: got %d want %d output=%s", exit, exitInvalidInput, stdout.String())
	}
}

func TestRegressRunExit5OnDrift(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeBaseline := filepath.Join(root, "baseline-store")
	storeCurrent := filepath.Join(root, "current-store")
	baselinePath := filepath.Join(root, "baseline.json")
	frameworkPath := fixturePath(t, "frameworks", "regress-minimal.yaml")
	recordPath := fixturePath(t, "records", "decision.json")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"record", "add", "--input", recordPath, "--store-dir", storeBaseline, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("record add baseline exit mismatch: got %d output=%s stderr=%s", exit, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	exit = execute([]string{"regress", "init", "--baseline", baselinePath, "--frameworks", frameworkPath, "--store-dir", storeBaseline, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("regress init exit mismatch: got %d output=%s stderr=%s", exit, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	exit = execute([]string{"regress", "run", "--baseline", baselinePath, "--frameworks", frameworkPath, "--store-dir", storeBaseline, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("regress run same store exit mismatch: got %d output=%s stderr=%s", exit, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	exit = execute([]string{"regress", "run", "--baseline", baselinePath, "--frameworks", frameworkPath, "--store-dir", storeCurrent, "--json"}, &stdout, &stderr)
	if exit != exitRegressionDrift {
		t.Fatalf("regress drift exit mismatch: got %d want %d output=%s stderr=%s", exit, exitRegressionDrift, stdout.String(), stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode output json: %v output=%s", err, stdout.String())
	}
	if payload["ok"] != false {
		t.Fatalf("expected ok=false output=%s", stdout.String())
	}
}

func fixturePath(t *testing.T, parts ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	base := append([]string{filepath.Dir(file), "..", "..", "fixtures"}, parts...)
	return filepath.Clean(filepath.Join(base...))
}
