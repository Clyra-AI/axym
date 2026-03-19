package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBundleJSONSuccess(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	outDir := filepath.Join(t.TempDir(), "bundle")
	fixtureDir := filepath.Join("fixtures", "collectors")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if exit := execute([]string{"collect", "--fixture-dir", fixtureDir, "--store-dir", storeDir, "--json"}, &stdout, &stderr); exit != exitSuccess {
		t.Fatalf("collect setup exit mismatch: got %d want %d stdout=%s stderr=%s", exit, exitSuccess, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	exit := execute([]string{
		"bundle",
		"--audit", "Q3-2026",
		"--frameworks", "eu-ai-act,soc2",
		"--store-dir", storeDir,
		"--output", outDir,
		"--json",
	}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("bundle exit mismatch: got %d want %d stdout=%s stderr=%s", exit, exitSuccess, stdout.String(), stderr.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload["command"] != "bundle" {
		t.Fatalf("command mismatch: got %v payload=%s", payload["command"], stdout.String())
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok=true payload=%s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(outDir, "manifest.json")); err != nil {
		t.Fatalf("missing manifest: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "oscal-v1.1", "component-definition.json")); err != nil {
		t.Fatalf("missing oscal export: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "identity-chain-summary.json")); err != nil {
		t.Fatalf("missing identity chain summary: %v", err)
	}
}

func TestBundleRejectsUnsafeOutputPath(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	outDir := filepath.Join(t.TempDir(), "bundle")
	if err := os.MkdirAll(outDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "foreign.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"bundle", "--audit", "Q3-2026", "--store-dir", storeDir, "--output", outDir, "--json"}, &stdout, &stderr)
	if exit != exitUnsafeBlocked {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s stderr=%s", exit, exitUnsafeBlocked, stdout.String(), stderr.String())
	}
}

func TestBundleRequiresAuditName(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"bundle", "--json"}, &stdout, &stderr)
	if exit != exitInvalidInput {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s stderr=%s", exit, exitInvalidInput, stdout.String(), stderr.String())
	}
}
