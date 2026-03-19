package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Clyra-AI/proof"
)

func TestInvalidEvidenceDoesNotCountTowardCoverage(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatalf("mkdir store: %v", err)
	}
	chain := proof.NewChain("test-invalid-evidence")
	invalidRecord := mustContractRecord(t, proof.RecordOpts{
		Timestamp:     time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC),
		Source:        "llmapi",
		SourceProduct: "axym",
		Type:          "decision",
		Event: map[string]any{
			"model":               "x",
			"decision":            "deny",
			"actor_identity":      "agent://requester",
			"downstream_identity": "agent://executor",
			"owner_identity":      "owner://payments",
			"policy_digest":       "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"target_kind":         "tool",
			"target_id":           "planner",
			"delegation_chain": []any{
				map[string]any{"identity": "agent://requester", "role": "requester"},
				map[string]any{"identity": "agent://executor", "role": "delegate"},
			},
		},
		Metadata: map[string]any{"reason_code": "schema_error"},
		Controls: proof.Controls{PermissionsEnforced: true},
	})
	validRecord := mustContractRecord(t, proof.RecordOpts{
		Timestamp:     time.Date(2026, 3, 1, 9, 0, 1, 0, time.UTC),
		Source:        "llmapi",
		SourceProduct: "axym",
		Type:          "decision",
		Event: map[string]any{
			"model":               "x",
			"decision":            "allow",
			"actor_identity":      "agent://requester",
			"downstream_identity": "agent://executor",
			"owner_identity":      "owner://payments",
			"policy_digest":       "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"target_kind":         "tool",
			"target_id":           "planner",
			"delegation_chain": []any{
				map[string]any{"identity": "agent://requester", "role": "requester"},
				map[string]any{"identity": "agent://executor", "role": "delegate"},
			},
		},
		Controls: proof.Controls{PermissionsEnforced: true},
	})
	if err := proof.AppendToChain(chain, &invalidRecord); err != nil {
		t.Fatalf("append invalid record: %v", err)
	}
	if err := proof.AppendToChain(chain, &validRecord); err != nil {
		t.Fatalf("append valid record: %v", err)
	}
	rawChain, err := json.MarshalIndent(chain, "", "  ")
	if err != nil {
		t.Fatalf("marshal chain: %v", err)
	}
	if err := os.WriteFile(filepath.Join(storeDir, "chain.json"), rawChain, 0o600); err != nil {
		t.Fatalf("write chain: %v", err)
	}

	frameworkPath := filepath.Join(root, "single-control-framework.yaml")
	frameworkYAML := []byte(`framework:
  id: single-control
  version: "1"
  title: Single Control Framework
controls:
  - id: c1
    title: Decision Evidence
    required_record_types: [decision]
    required_fields: [record_id, timestamp, source, source_product, record_type, event, integrity.record_hash]
    minimum_frequency: continuous
`)
	if err := os.WriteFile(frameworkPath, frameworkYAML, 0o600); err != nil {
		t.Fatalf("write framework: %v", err)
	}

	output, exit := runAxymContract(t, "map", "--frameworks", frameworkPath, "--store-dir", storeDir, "--json")
	if exit != 0 {
		t.Fatalf("map exit mismatch: %d output=%s", exit, output)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	frameworks, _ := data["frameworks"].([]any)
	if len(frameworks) != 1 {
		t.Fatalf("framework output mismatch: %s", output)
	}
	frameworkObj, _ := frameworks[0].(map[string]any)
	controls, _ := frameworkObj["controls"].([]any)
	if len(controls) != 1 {
		t.Fatalf("control output mismatch: %s", output)
	}
	control, _ := controls[0].(map[string]any)
	if control["matched_count"] != float64(1) {
		t.Fatalf("matched count mismatch (invalid evidence should not be counted as valid): %s", output)
	}
	if control["invalid_excluded"] != float64(1) {
		t.Fatalf("invalid exclusion mismatch: %s", output)
	}
}

func mustContractRecord(t *testing.T, opts proof.RecordOpts) proof.Record {
	t.Helper()
	record, err := proof.NewRecord(opts)
	if err != nil {
		t.Fatalf("proof.NewRecord: %v", err)
	}
	return *record
}
