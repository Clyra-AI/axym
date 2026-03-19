package grade

import (
	"testing"

	"github.com/Clyra-AI/axym/core/compliance/coverage"
	"github.com/Clyra-AI/axym/core/compliance/match"
)

func TestWeakestLinkDeterministic(t *testing.T) {
	t.Parallel()

	result := Derive(coverage.Report{Summary: coverage.Summary{
		ControlCount: 4,
		CoveredCount: 2,
		PartialCount: 1,
		GapCount:     1,
	}})
	if result.Letter != "C" {
		t.Fatalf("letter mismatch: %+v", result)
	}
	if result.Score != 0.625 {
		t.Fatalf("score mismatch: %+v", result)
	}
	if result.Reason == "" {
		t.Fatalf("expected rationale")
	}
}

func TestWeakestLinkAppliesIdentityPenalty(t *testing.T) {
	t.Parallel()

	result := Derive(coverage.Report{
		Frameworks: []coverage.FrameworkCoverage{{
			FrameworkID: "fw",
			Controls: []coverage.ControlCoverage{{
				FrameworkID: "fw",
				ControlID:   "c1",
				Status:      match.ControlStatusCovered,
				ReasonCodes: []string{match.ReasonMissingOwnerLinkage},
			}},
		}},
		Summary: coverage.Summary{
			ControlCount: 1,
			CoveredCount: 1,
		},
	})
	if result.Letter != "B" {
		t.Fatalf("expected identity penalty to cap A to B: %+v", result)
	}
	if result.Score >= 1 {
		t.Fatalf("expected penalized score below 1: %+v", result)
	}
}
