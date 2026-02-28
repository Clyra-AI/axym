package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestVersionJSONContract(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"version", "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d want %d stderr=%s", exit, exitSuccess, stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v output=%s", err, stdout.String())
	}
	if payload["command"] != "version" {
		t.Fatalf("command mismatch: got %v", payload["command"])
	}
	data, _ := payload["data"].(map[string]any)
	if _, ok := data["version"].(string); !ok {
		t.Fatalf("missing version field: output=%s", stdout.String())
	}
}

func TestVersionQuietSuppressesText(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"version", "--quiet"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d want %d stderr=%s", exit, exitSuccess, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("expected no stdout for --quiet, got %q", stdout.String())
	}
}
