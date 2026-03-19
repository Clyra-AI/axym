package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInitJSONCreatesPolicyAndStore(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	policyPath := filepath.Join(root, "axym-policy.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"init", "--store-dir", storeDir, "--policy-path", policyPath, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v output=%s", err, stdout.String())
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data envelope: %s", stdout.String())
	}
	if data["policy_created"] != true {
		t.Fatalf("expected policy_created=true output=%s", stdout.String())
	}
	if _, ok := data["sample_pack"]; ok {
		t.Fatalf("did not expect sample_pack data when flag omitted output=%s", stdout.String())
	}
	if _, err := os.Stat(policyPath); err != nil {
		t.Fatalf("stat policy file: %v", err)
	}
}

func TestInitJSONCreatesSamplePackWhenRequested(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	policyPath := filepath.Join(root, "axym-policy.yaml")
	samplePackDir := filepath.Join(root, "axym-sample")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{
		"init",
		"--store-dir", storeDir,
		"--policy-path", policyPath,
		"--sample-pack", samplePackDir,
		"--json",
	}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v output=%s", err, stdout.String())
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data envelope: %s", stdout.String())
	}
	samplePack, ok := data["sample_pack"].(map[string]any)
	if !ok {
		t.Fatalf("missing sample_pack output: %s", stdout.String())
	}
	if samplePack["path"] != samplePackDir {
		t.Fatalf("sample pack path mismatch: %v", samplePack["path"])
	}
	files, _ := samplePack["files"].([]any)
	if len(files) != 3 {
		t.Fatalf("expected 3 sample pack files output=%s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(samplePackDir, "governance", "context_engineering.jsonl")); err != nil {
		t.Fatalf("stat governance asset: %v", err)
	}
	if _, err := os.Stat(filepath.Join(samplePackDir, "records", "approval.json")); err != nil {
		t.Fatalf("stat approval asset: %v", err)
	}
	if _, err := os.Stat(filepath.Join(samplePackDir, "records", "risk_assessment.json")); err != nil {
		t.Fatalf("stat risk asset: %v", err)
	}
}

func TestInitInvalidExistingPolicyFailsClosed(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	policyPath := filepath.Join(root, "axym-policy.yaml")
	if err := os.WriteFile(policyPath, []byte("version: v1\ndefaults:\n  store_dir: .axym\n"), 0o600); err != nil {
		t.Fatalf("write invalid policy: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"init", "--store-dir", storeDir, "--policy-path", policyPath, "--json"}, &stdout, &stderr)
	if exit != exitInvalidInput {
		t.Fatalf("exit mismatch: got %d want %d output=%s", exit, exitInvalidInput, stdout.String())
	}
}

func TestInitRejectsInvalidSamplePackTarget(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	policyPath := filepath.Join(root, "axym-policy.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{
		"init",
		"--store-dir", storeDir,
		"--policy-path", policyPath,
		"--sample-pack", ".",
		"--json",
	}, &stdout, &stderr)
	if exit != exitInvalidInput {
		t.Fatalf("exit mismatch: got %d want %d output=%s", exit, exitInvalidInput, stdout.String())
	}
}
