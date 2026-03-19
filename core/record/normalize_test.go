package record

import (
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/normalize"
	"github.com/Clyra-AI/axym/core/redact"
)

func TestNormalizeAndBuildRejectsMalformedInput(t *testing.T) {
	t.Parallel()

	_, err := NormalizeAndBuild(normalize.Input{
		SourceType: "mcp",
		Event:      map[string]any{"wrong": true},
		Controls:   normalize.Controls{PermissionsEnforced: true},
	}, redact.Config{})
	if err == nil {
		t.Fatal("expected error for malformed input")
	}
	if ReasonCode(err) != ReasonMappingError {
		t.Fatalf("reason mismatch: got %q want %q", ReasonCode(err), ReasonMappingError)
	}
}

func TestNormalizeAndBuildDeterministicRedaction(t *testing.T) {
	t.Parallel()

	input := normalize.Input{
		SourceType:    "mcp",
		Source:        "mcp-runtime",
		SourceProduct: "axym",
		Timestamp:     time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC),
		Event: map[string]any{
			"tool_name": "query",
			"api_key":   "secret-value",
		},
		Controls: normalize.Controls{PermissionsEnforced: true},
	}
	rules := redact.Config{EventRules: []redact.Rule{{Path: "api_key", Action: redact.ActionHash}}}

	r1, err := NormalizeAndBuild(input, rules)
	if err != nil {
		t.Fatalf("first build error = %v", err)
	}
	r2, err := NormalizeAndBuild(input, rules)
	if err != nil {
		t.Fatalf("second build error = %v", err)
	}
	if r1.Event["api_key"] == "secret-value" {
		t.Fatal("expected api_key to be redacted")
	}
	if r1.Event["api_key"] != r2.Event["api_key"] {
		t.Fatalf("redacted hash mismatch: %v vs %v", r1.Event["api_key"], r2.Event["api_key"])
	}
}

func TestBuildRejectsSchemaInvalid(t *testing.T) {
	t.Parallel()

	normalized := normalize.Record{
		Source:        "mcp",
		SourceProduct: "",
		RecordType:    "tool_invocation",
		Timestamp:     time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC),
		Event:         map[string]any{"tool_name": "x"},
		Controls:      normalize.Controls{PermissionsEnforced: true},
	}
	_, err := Build(BuildInput{Normalized: normalized})
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	if ReasonCode(err) != ReasonSchemaError {
		t.Fatalf("reason mismatch: got %q want %q", ReasonCode(err), ReasonSchemaError)
	}
}

func TestBuildCanonicalizesEmptyMetadata(t *testing.T) {
	t.Parallel()

	normalized := normalize.Record{
		Source:        "mcp",
		SourceProduct: "axym",
		RecordType:    "tool_invocation",
		Timestamp:     time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC),
		Event:         map[string]any{"tool_name": "fetch"},
		Metadata:      map[string]any{},
		Controls:      normalize.Controls{PermissionsEnforced: true},
	}

	record, err := Build(BuildInput{Normalized: normalized})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if record.Metadata != nil {
		t.Fatalf("expected empty metadata to canonicalize to nil, got %#v", record.Metadata)
	}
}

func TestBuildSynthesizesIdentityRelationship(t *testing.T) {
	t.Parallel()

	normalized := normalize.Record{
		Source:        "governanceevent",
		SourceProduct: "axym",
		RecordType:    "policy_enforcement",
		AgentID:       "agent://executor",
		Timestamp:     time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC),
		Event: map[string]any{
			"governance_event_type": "policy_eval",
			"actor_identity":        "agent://requester",
			"downstream_identity":   "agent://executor",
			"owner_identity":        "owner://payments",
			"policy_digest":         "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"approval_token_ref":    "approval://chg-123",
			"target_kind":           "tool",
			"target_id":             "db.query",
			"delegation_chain": []any{
				map[string]any{"identity": "agent://requester", "role": "requester"},
				map[string]any{"identity": "agent://executor", "role": "delegate"},
			},
		},
		Controls: normalize.Controls{PermissionsEnforced: true},
	}

	record, err := Build(BuildInput{Normalized: normalized})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if record.Relationship == nil {
		t.Fatalf("expected relationship synthesis")
	}
	if record.Relationship.PolicyRef == nil || record.Relationship.PolicyRef.PolicyDigest == "" {
		t.Fatalf("expected policy digest relationship: %+v", record.Relationship)
	}
	if len(record.Relationship.AgentChain) != 2 {
		t.Fatalf("expected delegation chain relationship: %+v", record.Relationship)
	}
}
