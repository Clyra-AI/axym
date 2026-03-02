package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRootJSONEnvelope(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s", exit, stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode output json: %v output=%s", err, stdout.String())
	}
	if payload["ok"] != true {
		t.Fatalf("expected ok=true: %s", stdout.String())
	}
	if payload["command"] != "root" {
		t.Fatalf("command mismatch: %s", stdout.String())
	}
}

func TestRootQuietSuppressesText(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"--quiet"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s", exit, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("expected quiet output, got %q", stdout.String())
	}
}
