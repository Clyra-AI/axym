package gaps

import (
	"encoding/json"
	"testing"

	"github.com/Clyra-AI/axym/core/compliance/coverage"
)

func TestRankingDeterminism(t *testing.T) {
	t.Parallel()

	report := coverage.Report{Frameworks: []coverage.FrameworkCoverage{
		{
			FrameworkID: "soc2",
			Controls: []coverage.ControlCoverage{
				{FrameworkID: "soc2", ControlID: "cc7", Title: "System Operations", Status: "gap", MissingRecordTypes: []string{"incident", "guardrail_activation"}, MissingFields: []string{"integrity.record_hash"}},
				{FrameworkID: "soc2", ControlID: "cc8", Title: "Change Management", Status: "partial", MissingFields: []string{"record_id"}},
			},
		},
		{
			FrameworkID: "eu-ai-act",
			Controls: []coverage.ControlCoverage{
				{FrameworkID: "eu-ai-act", ControlID: "article-9", Title: "Risk Management", Status: "gap", MissingRecordTypes: []string{"risk_assessment"}},
			},
		},
	}, Summary: coverage.Summary{ControlCount: 3, GapCount: 2, PartialCount: 1}}

	first := Build(report)
	second := Build(report)
	left, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first: %v", err)
	}
	right, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second: %v", err)
	}
	if string(left) != string(right) {
		t.Fatalf("ranking must be stable:\n%s\n%s", string(left), string(right))
	}
	if len(first.Gaps) != 3 {
		t.Fatalf("gap count mismatch: %+v", first)
	}
	if first.Gaps[0].Rank != 1 || first.Gaps[0].Status != "gap" {
		t.Fatalf("unexpected first ranked item: %+v", first.Gaps[0])
	}
}
