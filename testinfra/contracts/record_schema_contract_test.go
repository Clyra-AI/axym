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
		"event": {"tool_name": "fetch"},
		"metadata": {"env": "ci"},
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
