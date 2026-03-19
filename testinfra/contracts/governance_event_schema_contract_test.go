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
		"downstream_identity":"agent://executor",
		"owner_identity":"owner://payments",
		"policy_digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"approval_token_ref":"approval://chg-123",
		"delegation_chain":[{"identity":"agent://requester","role":"requester"},{"identity":"agent://executor","role":"delegate"}],
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

func TestGovernanceEventSchemaContract_ContextEngineering(t *testing.T) {
	t.Parallel()

	valid := []byte(`{
		"event_type":"instruction_rewrite",
		"source":"agent-fw",
		"timestamp":"2026-03-18T12:00:00Z",
		"actor":{"id":"agent-1","type":"agent"},
		"action":"rewrite",
		"target":{"kind":"instruction_set","id":"system-prompt"},
		"context":{
			"previous_hash":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"current_hash":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"artifact_kind":"instruction_set",
			"reason_code":"POLICY_REFRESH"
		}
	}`)
	if err := governanceeventschema.Validate(valid); err != nil {
		t.Fatalf("valid context event rejected: %v", err)
	}

	invalid := []byte(`{
		"event_type":"knowledge_import",
		"source":"agent-fw",
		"timestamp":"2026-03-18T12:00:00Z",
		"actor":{"id":"agent-1","type":"agent"},
		"action":"import",
		"target":{"kind":"knowledge_artifact","id":"kb:policy-pack"},
		"context":{
			"artifact_kind":"knowledge_artifact",
			"source_uri":"repo://policy/pack"
		},
		"metadata":{"knowledge_body":"raw content should be rejected"}
	}`)
	if err := governanceeventschema.Validate(invalid); err == nil {
		t.Fatal("invalid context event accepted")
	}
}
