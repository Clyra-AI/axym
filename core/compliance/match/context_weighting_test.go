package match

import (
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/compliance/framework"
	"github.com/Clyra-AI/proof"
)

func TestContextWeightingDeterministicRationale(t *testing.T) {
	t.Parallel()

	defs := []framework.Definition{
		{
			ID:      "test-fw",
			Version: "v1",
			Title:   "Test",
			Controls: []framework.Control{
				{
					FrameworkID:         "test-fw",
					ID:                  "c1",
					Title:               "Control 1",
					RequiredRecordTypes: []string{"tool_invocation"},
					RequiredFields:      []string{"record_id", "timestamp", "source", "event", "integrity.record_hash"},
					MinimumFrequency:    "continuous",
				},
			},
		},
	}

	records := []proof.Record{
		mustRecord(t, proof.RecordOpts{
			Timestamp:     time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
			Source:        "mcp",
			SourceProduct: "axym",
			Type:          "tool_invocation",
			Event:         map[string]any{"tool_name": "filesystem.write"},
			Metadata: map[string]any{
				"data_class":       "restricted",
				"risk_level":       "high",
				"discovery_method": "runtime",
			},
			Controls: proof.Controls{PermissionsEnforced: true},
		}),
		mustRecord(t, proof.RecordOpts{
			Timestamp:     time.Date(2026, 3, 1, 12, 0, 1, 0, time.UTC),
			Source:        "mcp",
			SourceProduct: "axym",
			Type:          "tool_invocation",
			Event:         map[string]any{"tool_name": "filesystem.read"},
			Metadata:      map[string]any{"data_class": "public", "risk_level": "low", "discovery_method": "runtime"},
			Controls:      proof.Controls{PermissionsEnforced: true},
		}),
	}

	result := Evaluate(defs, records, Options{ExcludeInvalidEvidence: true})
	if len(result.Frameworks) != 1 || len(result.Frameworks[0].Controls) != 1 {
		t.Fatalf("unexpected result shape: %+v", result)
	}
	control := result.Frameworks[0].Controls[0]
	if control.Status != ControlStatusCovered {
		t.Fatalf("status mismatch: %+v", control)
	}
	if len(control.Evidence) != 2 {
		t.Fatalf("evidence mismatch: %+v", control)
	}
	if control.Evidence[0].Weights.Total <= control.Evidence[1].Weights.Total {
		t.Fatalf("expected higher contextual weight for restricted/high evidence: %+v", control.Evidence)
	}
	if control.Evidence[0].ReasonCodes[0] != "MATCHED" {
		t.Fatalf("expected match reason: %+v", control.Evidence[0])
	}
	if control.Rationale == "" {
		t.Fatalf("expected rationale")
	}
}

func mustRecord(t *testing.T, opts proof.RecordOpts) proof.Record {
	t.Helper()
	record, err := proof.NewRecord(opts)
	if err != nil {
		t.Fatalf("proof.NewRecord: %v", err)
	}
	return *record
}
