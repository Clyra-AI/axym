package governanceevent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Clyra-AI/axym/core/collector"
)

func TestGovernanceEventPromotion(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "events.jsonl")
	content := []byte(`{"event_type":"policy_eval","source":"agent-fw","timestamp":"2026-02-28T12:00:00Z","actor":{"id":"agent-1","type":"agent"},"action":"tool_call","target":{"kind":"tool","id":"db.query"},"metadata":{"risk":"high"}}` + "\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	result, err := Collector{}.Collect(context.Background(), collector.Request{GovernanceEventFiles: []string{path}})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidate count mismatch: %+v", result)
	}
	candidate := result.Candidates[0]
	if candidate.RecordType != "policy_enforcement" {
		t.Fatalf("record type mismatch: %+v", candidate)
	}
	if candidate.Event["governance_source"] != "agent-fw" || candidate.Event["actor_id"] != "agent-1" {
		t.Fatalf("missing governance provenance fields: %+v", candidate.Event)
	}
}

func TestGovernanceEventRejectsMalformedSchema(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "events.jsonl")
	content := []byte(`{"source":"agent-fw"}` + "\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := Collector{}.Collect(context.Background(), collector.Request{GovernanceEventFiles: []string{path}})
	if err == nil {
		t.Fatal("expected schema rejection")
	}
	if rc, ok := err.(interface{ ReasonCode() string }); !ok || rc.ReasonCode() != ReasonSchemaError {
		t.Fatalf("reason mismatch: err=%v", err)
	}
}

func TestGovernanceEventPromotion_ContextEngineering(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "events.jsonl")
	content := []byte(`{"event_type":"knowledge_import","source":"agent-fw","timestamp":"2026-03-18T12:00:00Z","actor":{"id":"agent-ctx","type":"agent"},"action":"import","target":{"kind":"knowledge_artifact","id":"kb:policy-pack"},"context":{"artifact_digest":"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","artifact_kind":"knowledge_artifact","source_uri":"repo://policy/pack","reason_code":"KNOWLEDGE_SYNC","approval_ref":"chg-42"}}` + "\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	result, err := Collector{}.Collect(context.Background(), collector.Request{GovernanceEventFiles: []string{path}})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidate count mismatch: %+v", result)
	}
	candidate := result.Candidates[0]
	if candidate.RecordType != "decision" {
		t.Fatalf("record type mismatch: %+v", candidate)
	}
	if candidate.Event["context_event_class"] != "context_engineering" {
		t.Fatalf("missing context event class: %+v", candidate.Event)
	}
	if candidate.Event["context_artifact_digest"] != "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc" {
		t.Fatalf("missing artifact digest: %+v", candidate.Event)
	}
	if candidate.Event["context_reason_code"] != "KNOWLEDGE_SYNC" {
		t.Fatalf("missing reason code: %+v", candidate.Event)
	}
}
