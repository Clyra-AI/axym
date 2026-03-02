package axym

import (
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/compliance/framework"
	"github.com/Clyra-AI/axym/core/compliance/match"
	"github.com/Clyra-AI/proof"
)

func TestScenarioContextWeightedControlMapping(t *testing.T) {
	t.Parallel()

	defs := []framework.Definition{{
		ID:      "scenario-fw",
		Version: "1",
		Title:   "Scenario",
		Controls: []framework.Control{{
			FrameworkID:         "scenario-fw",
			ID:                  "control-1",
			Title:               "Control",
			RequiredRecordTypes: []string{"decision"},
			RequiredFields:      []string{"record_id", "timestamp", "source", "event", "integrity.record_hash"},
			MinimumFrequency:    "continuous",
		}},
	}}

	high := mustScenarioRecord(t, proof.RecordOpts{
		Timestamp:     time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
		Source:        "llmapi",
		SourceProduct: "axym",
		Type:          "decision",
		Event:         map[string]any{"model": "x", "decision": "allow"},
		Metadata:      map[string]any{"risk_level": "high", "data_class": "restricted", "discovery_method": "runtime"},
		Controls:      proof.Controls{PermissionsEnforced: true},
	})
	low := mustScenarioRecord(t, proof.RecordOpts{
		Timestamp:     time.Date(2026, 3, 1, 10, 0, 1, 0, time.UTC),
		Source:        "llmapi",
		SourceProduct: "axym",
		Type:          "decision",
		Event:         map[string]any{"model": "x", "decision": "allow"},
		Metadata:      map[string]any{"risk_level": "low", "data_class": "public", "discovery_method": "ingest"},
		Controls:      proof.Controls{PermissionsEnforced: true},
	})

	result := match.Evaluate(defs, []proof.Record{high, low}, match.Options{ExcludeInvalidEvidence: true})
	control := result.Frameworks[0].Controls[0]
	if control.Evidence[0].Weights.Total <= control.Evidence[1].Weights.Total {
		t.Fatalf("expected high context to carry stronger weight: %+v", control.Evidence)
	}
}

func mustScenarioRecord(t *testing.T, opts proof.RecordOpts) proof.Record {
	t.Helper()
	record, err := proof.NewRecord(opts)
	if err != nil {
		t.Fatalf("proof.NewRecord: %v", err)
	}
	return *record
}
