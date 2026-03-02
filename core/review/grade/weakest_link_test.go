package grade

import (
	"testing"

	"github.com/Clyra-AI/axym/core/compliance/coverage"
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
