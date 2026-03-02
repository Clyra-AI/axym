package contracts

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestReviewJSONEnvelopeContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	stdout, exit := runAxymContract(t, "review", "--date", "2026-09-15", "--store-dir", storeDir, "--json")
	if exit != 0 {
		t.Fatalf("unexpected exit %d output=%s", exit, stdout)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode output json: %v", err)
	}
	if payload["command"] != "review" {
		t.Fatalf("command mismatch: got=%v", payload["command"])
	}
	data, _ := payload["data"].(map[string]any)
	if _, ok := data["exceptions"].([]any); !ok {
		t.Fatalf("missing exception classes: %s", stdout)
	}
}

func TestOverrideAndReplayJSONEnvelopeContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	overrideOut, overrideExit := runAxymContract(t, "override", "create", "--bundle", "Q3-2026", "--reason", "fixture", "--signer", "ops-key", "--store-dir", storeDir, "--json")
	if overrideExit != 0 {
		t.Fatalf("override exit mismatch: %d output=%s", overrideExit, overrideOut)
	}
	var overridePayload map[string]any
	if err := json.Unmarshal([]byte(overrideOut), &overridePayload); err != nil {
		t.Fatalf("decode override output: %v", err)
	}
	if overridePayload["command"] != "override" {
		t.Fatalf("override command mismatch: %s", overrideOut)
	}

	replayOut, replayExit := runAxymContract(t, "replay", "--model", "payments-agent", "--tier", "A", "--store-dir", storeDir, "--json")
	if replayExit != 0 {
		t.Fatalf("replay exit mismatch: %d output=%s", replayExit, replayOut)
	}
	var replayPayload map[string]any
	if err := json.Unmarshal([]byte(replayOut), &replayPayload); err != nil {
		t.Fatalf("decode replay output: %v", err)
	}
	if replayPayload["command"] != "replay" {
		t.Fatalf("replay command mismatch: %s", replayOut)
	}
}
