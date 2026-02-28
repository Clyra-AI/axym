package contracts

import (
	"testing"

	governanceeventschema "github.com/Clyra-AI/axym/schemas/v1/governance_event"
)

func TestGovernanceEventSchemaContract(t *testing.T) {
	t.Parallel()

	valid := []byte(`{
		"event_type":"policy_eval",
		"source":"agent-fw",
		"timestamp":"2026-02-28T12:00:00Z",
		"actor":{"id":"agent-1","type":"agent"},
		"action":"tool_call",
		"target":{"kind":"tool","id":"db.query"}
	}`)
	if err := governanceeventschema.Validate(valid); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}

	invalid := []byte(`{
		"source":"agent-fw",
		"timestamp":"2026-02-28T12:00:00Z"
	}`)
	if err := governanceeventschema.Validate(invalid); err == nil {
		t.Fatal("invalid event accepted")
	}
}
