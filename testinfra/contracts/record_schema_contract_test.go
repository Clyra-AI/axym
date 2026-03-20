package contracts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	recordschema "github.com/Clyra-AI/axym/schemas/v1/record"
)

func TestNormalizedRecordSchemaContract(t *testing.T) {
	t.Parallel()

	valid := []byte(`{
		"source": "mcp",
		"source_product": "axym",
		"record_type": "tool_invocation",
		"timestamp": "2026-02-28T12:00:00Z",
		"event": {
			"tool_name": "fetch",
			"actor_identity": "agent://requester",
			"downstream_identity": "agent://executor",
			"owner_identity": "owner://payments",
			"policy_digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"approval_token_ref": "approval://chg-123",
			"target_kind": "tool",
			"target_id": "fetch"
		},
		"metadata": {"env": "ci"},
		"relationship": {
			"policy_ref": {"policy_digest": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
		},
		"controls": {"permissions_enforced": true}
	}`)
	if err := recordschema.ValidateNormalized(valid); err != nil {
		t.Fatalf("valid schema payload rejected: %v", err)
	}

	invalid := []byte(`{
		"source": "mcp",
		"record_type": "tool_invocation",
		"timestamp": "2026-02-28T12:00:00Z",
		"event": {"tool_name": "fetch"},
		"controls": {"permissions_enforced": true}
	}`)
	if err := recordschema.ValidateNormalized(invalid); err == nil {
		t.Fatal("invalid payload accepted; source_product should be required")
	}
}

func TestRecordAddRejectsUnknownTypeContract(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	recordPath := filepath.Join(root, "record.json")
	payload := []byte(`{
  "record_id": "contract-unknown-type",
  "record_version": "v1",
  "timestamp": "2026-03-18T00:00:00Z",
  "source": "manual",
  "source_product": "axym",
  "agent_id": "agent-1",
  "record_type": "does_not_exist",
  "event": {"anything": true}
}`)
	if err := os.WriteFile(recordPath, payload, 0o600); err != nil {
		t.Fatalf("write record fixture: %v", err)
	}
	out, exit := runAxymContract(t, "record", "add", "--input", recordPath, "--store-dir", filepath.Join(root, "store"), "--json")
	if exit != 3 {
		t.Fatalf("exit mismatch: got %d want 3 output=%s", exit, out)
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(out), &envelope); err != nil {
		t.Fatalf("decode output: %v output=%s", err, out)
	}
	errObj, _ := envelope["error"].(map[string]any)
	if errObj["reason"] != "schema_violation" {
		t.Fatalf("reason mismatch: got %v output=%s", errObj["reason"], out)
	}
}
