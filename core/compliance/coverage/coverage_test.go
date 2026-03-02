package coverage

import (
	"testing"

	"github.com/Clyra-AI/axym/core/compliance/match"
)

func TestBuildCoverageStableShape(t *testing.T) {
	t.Parallel()

	report := Build(match.Result{Frameworks: []match.FrameworkResult{{
		ID: "fw",
		Controls: []match.ControlResult{
			{
				FrameworkID:         "fw",
				ControlID:           "c1",
				Title:               "Control",
				Status:              match.ControlStatusPartial,
				ReasonCodes:         []string{"CONTROL_PARTIAL"},
				RequiredRecordTypes: []string{"decision"},
				Evidence: []match.RecordMatch{{
					RecordID:   "r1",
					RecordType: "decision",
					Missing:    []string{"integrity.record_hash"},
				}},
			},
		},
	}}})

	if len(report.Frameworks) != 1 {
		t.Fatalf("framework count mismatch: %+v", report)
	}
	control := report.Frameworks[0].Controls[0]
	if control.Status != match.ControlStatusPartial {
		t.Fatalf("status mismatch: %+v", control)
	}
	if len(control.MissingFields) != 1 || control.MissingFields[0] != "integrity.record_hash" {
		t.Fatalf("missing fields mismatch: %+v", control)
	}
}
