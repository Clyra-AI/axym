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
