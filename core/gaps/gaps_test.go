package gaps

import (
	"testing"

	"github.com/Clyra-AI/axym/core/compliance/coverage"
)

func TestExplainIncludesGradeAndRankedItems(t *testing.T) {
	t.Parallel()

	report := Build(coverage.Report{Frameworks: []coverage.FrameworkCoverage{{
		FrameworkID: "fw",
		Controls: []coverage.ControlCoverage{{
			FrameworkID: "fw",
			ControlID:   "c1",
			Title:       "Control 1",
			Status:      "gap",
		}},
	}}, Summary: coverage.Summary{ControlCount: 1, GapCount: 1}})

	lines := Explain(report)
	if len(lines) < 2 {
		t.Fatalf("expected explain lines: %+v", lines)
	}
}
