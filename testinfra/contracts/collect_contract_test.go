package contracts

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestCollectJSONEnvelopeContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	stdout, exit := runAxymContract(t, "collect", "--dry-run", "--fixture-dir", filepath.Join(testRepoRoot(t), "fixtures", "collectors"), "--store-dir", storeDir, "--json")
	if exit != 0 {
		t.Fatalf("unexpected exit %d output=%s", exit, stdout)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload["command"] != "collect" {
		t.Fatalf("command mismatch: got %v", payload["command"])
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data envelope: %s", stdout)
	}
	if _, ok := data["sources"].([]any); !ok {
		t.Fatalf("missing sources summary: %s", stdout)
	}
	if _, ok := data["reason_codes"]; !ok {
		t.Fatalf("missing reason_codes: %s", stdout)
	}
}

func TestCollectGovernanceContextEngineeringContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	governancePath := filepath.Join(testRepoRoot(t), "fixtures", "governance", "context_engineering.jsonl")
	stdout, exit := runAxymContract(t, "collect", "--governance-event-file", governancePath, "--store-dir", storeDir, "--json")
	if exit != 0 {
		t.Fatalf("unexpected exit %d output=%s", exit, stdout)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data envelope: %s", stdout)
	}
	if captured, _ := data["captured"].(float64); captured != 3 {
		t.Fatalf("captured mismatch: %s", stdout)
	}
	sources, ok := data["sources"].([]any)
	if !ok {
		t.Fatalf("missing sources summary: %s", stdout)
	}
	for _, candidate := range sources {
		source, _ := candidate.(map[string]any)
		if source["name"] != "governanceevent" {
			continue
		}
		if source["captured"] != float64(3) {
			t.Fatalf("governance source captured mismatch: %v", source)
		}
		if source["status"] != "ok" {
			t.Fatalf("governance source status mismatch: %v", source)
		}
		return
	}
	t.Fatalf("governanceevent source summary missing: %s", stdout)
}
