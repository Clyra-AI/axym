package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestReplayJSONContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"replay", "--model", "payments-agent", "--tier", "A", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got=%d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload["command"] != "replay" {
		t.Fatalf("command mismatch: %s", stdout.String())
	}
	data, _ := payload["data"].(map[string]any)
	if data["tier"] != "A" {
		t.Fatalf("tier mismatch: %s", stdout.String())
	}
	if _, ok := data["blast_radius"].(map[string]any); !ok {
		t.Fatalf("missing blast radius: %s", stdout.String())
	}
}

func TestReplayInvalidTierExitCode(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"replay", "--model", "payments-agent", "--tier", "Z", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitInvalidInput {
		t.Fatalf("exit mismatch: got=%d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
}
