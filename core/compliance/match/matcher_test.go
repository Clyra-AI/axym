package match

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/compliance/framework"
	"github.com/Clyra-AI/proof"
)

func TestEvaluateExcludesInvalidEvidenceFromCoverage(t *testing.T) {
	t.Parallel()

	defs := []framework.Definition{{
		ID:      "fw",
		Version: "v1",
		Title:   "Framework",
		Controls: []framework.Control{{
			FrameworkID:         "fw",
			ID:                  "c1",
			Title:               "Control",
			RequiredRecordTypes: []string{"decision"},
			RequiredFields:      []string{"record_id", "timestamp", "event", "integrity.record_hash"},
			MinimumFrequency:    "continuous",
		}},
	}}

	valid := mustMatchRecord(t, proof.RecordOpts{
		Timestamp:     time.Date(2026, 3, 1, 12, 0, 1, 0, time.UTC),
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
	invalid := mustMatchRecord(t, proof.RecordOpts{
		Timestamp:     time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
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

	result := Evaluate(defs, []proof.Record{valid, invalid}, Options{ExcludeInvalidEvidence: true})
	control := result.Frameworks[0].Controls[0]
	if control.Status != ControlStatusCovered {
		t.Fatalf("expected covered status with valid evidence: %+v", control)
	}
	if control.InvalidExcluded != 1 {
		t.Fatalf("invalid exclusion mismatch: %+v", control)
	}
}

func TestEvaluateInvalidEvidencePreservesActualMissingFields(t *testing.T) {
	t.Parallel()

	defs := []framework.Definition{{
		ID:      "fw",
		Version: "v1",
		Title:   "Framework",
		Controls: []framework.Control{{
			FrameworkID:         "fw",
			ID:                  "c1",
			Title:               "Control",
			RequiredRecordTypes: []string{"decision"},
			RequiredFields:      []string{"record_id", "timestamp", "event", "integrity.record_hash", "event.model", "event.decision"},
			MinimumFrequency:    "continuous",
		}},
	}}

	invalid := mustMatchRecord(t, proof.RecordOpts{
		Timestamp:     time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
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

	result := Evaluate(defs, []proof.Record{invalid}, Options{ExcludeInvalidEvidence: true})
	control := result.Frameworks[0].Controls[0]
	if control.InvalidExcluded != 1 {
		t.Fatalf("invalid exclusion mismatch: %+v", control)
	}
	if len(control.Evidence) != 1 {
		t.Fatalf("evidence count mismatch: %+v", control)
	}
	if len(control.Evidence[0].Missing) != 0 {
		t.Fatalf("unexpected missing fields for fully populated invalid record: %+v", control.Evidence[0].Missing)
	}
	if len(control.Evidence[0].Matched) == 0 {
		t.Fatalf("expected matched fields for invalid record evidence")
	}
}

func TestEvaluateDeterministicOutput(t *testing.T) {
	t.Parallel()

	defs := []framework.Definition{{
		ID:      "fw",
		Version: "v1",
		Title:   "Framework",
		Controls: []framework.Control{{
			FrameworkID:         "fw",
			ID:                  "c1",
			Title:               "Control",
			RequiredRecordTypes: []string{"tool_invocation"},
			RequiredFields:      []string{"record_id", "timestamp", "source", "event", "integrity.record_hash"},
			MinimumFrequency:    "continuous",
		}},
	}}

	records := []proof.Record{
		mustMatchRecord(t, proof.RecordOpts{
			Timestamp:     time.Date(2026, 3, 1, 12, 0, 2, 0, time.UTC),
			Source:        "mcp",
			SourceProduct: "axym",
			Type:          "tool_invocation",
			AgentID:       "agent-mcp",
			Event:         map[string]any{"tool_name": "filesystem.read"},
			Controls:      proof.Controls{PermissionsEnforced: true},
		}),
		mustMatchRecord(t, proof.RecordOpts{
			Timestamp:     time.Date(2026, 3, 1, 12, 0, 1, 0, time.UTC),
			Source:        "mcp",
			SourceProduct: "axym",
			Type:          "tool_invocation",
			AgentID:       "agent-mcp",
			Event:         map[string]any{"tool_name": "filesystem.write"},
			Controls:      proof.Controls{PermissionsEnforced: true},
		}),
	}

	first := Evaluate(defs, records, Options{ExcludeInvalidEvidence: true})
	second := Evaluate(defs, records, Options{ExcludeInvalidEvidence: true})
	left, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first: %v", err)
	}
	right, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second: %v", err)
	}
	if string(left) != string(right) {
		t.Fatalf("determinism mismatch:\n%s\n%s", string(left), string(right))
	}
}

func TestEvaluateDowngradesWeakIdentityLinkage(t *testing.T) {
	t.Parallel()

	defs := []framework.Definition{{
		ID:      "fw",
		Version: "v1",
		Title:   "Framework",
		Controls: []framework.Control{{
			FrameworkID:         "fw",
			ID:                  "c1",
			Title:               "Governed Decision",
			RequiredRecordTypes: []string{"decision"},
			RequiredFields:      []string{"record_id", "timestamp", "event", "integrity.record_hash"},
			MinimumFrequency:    "continuous",
		}},
	}}

	weak := mustMatchRecord(t, proof.RecordOpts{
		Timestamp:     time.Date(2026, 3, 1, 13, 0, 0, 0, time.UTC),
		Source:        "llmapi",
		SourceProduct: "axym",
		Type:          "decision",
		AgentID:       "agent://executor",
		Event:         map[string]any{"model": "x", "decision": "allow"},
		Controls:      proof.Controls{PermissionsEnforced: true},
	})

	result := Evaluate(defs, []proof.Record{weak}, Options{ExcludeInvalidEvidence: true})
	control := result.Frameworks[0].Controls[0]
	if control.Status != ControlStatusPartial {
		t.Fatalf("expected partial status for weak identity linkage: %+v", control)
	}
	if len(control.Evidence) != 1 {
		t.Fatalf("evidence count mismatch: %+v", control)
	}
	if !containsString(control.Evidence[0].ReasonCodes, ReasonMissingOwnerLinkage) {
		t.Fatalf("expected missing owner reason code: %+v", control.Evidence[0].ReasonCodes)
	}
	if !containsString(control.Evidence[0].Missing, "event.owner_identity") {
		t.Fatalf("expected missing owner field: %+v", control.Evidence[0].Missing)
	}
}

func mustMatchRecord(t *testing.T, opts proof.RecordOpts) proof.Record {
	t.Helper()
	record, err := proof.NewRecord(opts)
	if err != nil {
		t.Fatalf("proof.NewRecord: %v", err)
	}
	return *record
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
