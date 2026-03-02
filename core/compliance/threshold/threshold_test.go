package threshold

import (
	"testing"
	"time"

	"github.com/Clyra-AI/proof"
)

func TestEvaluateDeterministicFailures(t *testing.T) {
	t.Parallel()

	global := 0.70
	result := Evaluate([]FrameworkCoverage{
		{FrameworkID: "soc2", Coverage: 0.40, FailingControls: []string{"cc7", "cc6"}},
		{FrameworkID: "eu-ai-act", Coverage: 0.80},
	}, PolicyConfig{GlobalMinCoverage: &global}, nil)
	if result.Passed {
		t.Fatalf("expected failure: %+v", result)
	}
	if len(result.Failures) != 1 {
		t.Fatalf("failure count mismatch: %+v", result)
	}
	if result.Failures[0].FrameworkID != "soc2" {
		t.Fatalf("framework mismatch: %+v", result.Failures[0])
	}
	if result.Failures[0].FailingControls[0] != "cc6" {
		t.Fatalf("controls must be sorted: %+v", result.Failures[0].FailingControls)
	}
}

func TestIsInvalidEvidenceClass(t *testing.T) {
	t.Parallel()

	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
		Source:        "axym",
		SourceProduct: "axym",
		Type:          "decision",
		Event:         map[string]any{"model": "x"},
		Metadata: map[string]any{
			"reason_code": "schema_error",
		},
		Controls: proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("proof.NewRecord: %v", err)
	}

	invalid, code := IsInvalidEvidenceClass(*record)
	if !invalid {
		t.Fatalf("expected invalid evidence class")
	}
	if code != "schema_error" {
		t.Fatalf("code mismatch: %s", code)
	}
}
