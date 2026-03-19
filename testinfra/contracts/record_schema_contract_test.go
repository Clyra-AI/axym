package contracts

import (
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
