package regress

import (
	"path/filepath"
	"testing"

	"github.com/Clyra-AI/axym/core/compliance/coverage"
)

func TestInitAndRunNoDrift(t *testing.T) {
	t.Parallel()

	report := fixtureCoverage("covered")
	baselinePath := filepath.Join(t.TempDir(), "baseline.json")
	if _, err := Init(InitRequest{BaselinePath: baselinePath, CoverageReport: report}); err != nil {
		t.Fatalf("init baseline: %v", err)
	}
	result, err := Run(RunRequest{BaselinePath: baselinePath, CoverageReport: report})
	if err != nil {
		t.Fatalf("run baseline: %v", err)
	}
	if result.DriftDetected {
		t.Fatalf("expected no drift: %+v", result)
	}
}

func TestRunDetectsDeterministicDriftOrdering(t *testing.T) {
	t.Parallel()

	baselinePath := filepath.Join(t.TempDir(), "baseline.json")
	baselineReport := fixtureCoverage("covered")
	if _, err := Init(InitRequest{BaselinePath: baselinePath, CoverageReport: baselineReport}); err != nil {
		t.Fatalf("init baseline: %v", err)
	}
	driftReport := fixtureCoverage("gap")
	result, err := Run(RunRequest{BaselinePath: baselinePath, CoverageReport: driftReport})
	if err != nil {
		t.Fatalf("run baseline: %v", err)
	}
	if !result.DriftDetected {
		t.Fatalf("expected drift detected: %+v", result)
	}
	if len(result.RegressedControls) != 1 {
		t.Fatalf("unexpected drift count: %+v", result.RegressedControls)
	}
	if result.RegressedControls[0].ControlID != "A.1" {
		t.Fatalf("control ordering mismatch: %+v", result.RegressedControls)
	}
}

func fixtureCoverage(status string) coverage.Report {
	return coverage.Report{
		Frameworks: []coverage.FrameworkCoverage{
			{
				FrameworkID: "eu-ai-act",
				Coverage:    0.5,
				Controls: []coverage.ControlCoverage{
					{
						FrameworkID: "eu-ai-act",
						ControlID:   "A.1",
						Status:      status,
						ReasonCodes: []string{"CONTROL_" + status},
					},
				},
			},
		},
	}
}
