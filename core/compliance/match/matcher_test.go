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

	valid := mustRecord(t, proof.RecordOpts{
		Timestamp:     time.Date(2026, 3, 1, 12, 0, 1, 0, time.UTC),
		Source:        "llmapi",
		SourceProduct: "axym",
		Type:          "decision",
		Event:         map[string]any{"model": "x", "decision": "allow"},
		Controls:      proof.Controls{PermissionsEnforced: true},
	})
	invalid := mustRecord(t, proof.RecordOpts{
		Timestamp:     time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		Source:        "llmapi",
		SourceProduct: "axym",
		Type:          "decision",
		Event:         map[string]any{"model": "x", "decision": "deny"},
		Metadata:      map[string]any{"reason_code": "schema_error"},
		Controls:      proof.Controls{PermissionsEnforced: true},
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
		mustRecord(t, proof.RecordOpts{
			Timestamp:     time.Date(2026, 3, 1, 12, 0, 2, 0, time.UTC),
			Source:        "mcp",
			SourceProduct: "axym",
			Type:          "tool_invocation",
			Event:         map[string]any{"tool_name": "filesystem.read"},
			Controls:      proof.Controls{PermissionsEnforced: true},
		}),
		mustRecord(t, proof.RecordOpts{
			Timestamp:     time.Date(2026, 3, 1, 12, 0, 1, 0, time.UTC),
			Source:        "mcp",
			SourceProduct: "axym",
			Type:          "tool_invocation",
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
