package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestOverrideCreateJSON(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"override", "create", "--bundle", "Q3-2026", "--reason", "fixture", "--signer", "ops-key", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got=%d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload["command"] != "override" {
		t.Fatalf("command mismatch: %s", stdout.String())
	}
	data, _ := payload["data"].(map[string]any)
	if data["record_id"] == "" {
		t.Fatalf("missing record_id: %s", stdout.String())
	}
}

func TestOverrideMissingFlagsExitCode(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"override", "create", "--bundle", "Q3-2026", "--reason", "fixture", "--json"}, &stdout, &stderr)
	if exit != exitInvalidInput {
		t.Fatalf("exit mismatch: got=%d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
}
