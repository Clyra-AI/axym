package main

import (
	"fmt"
	"strings"

	"github.com/Clyra-AI/axym/core/compliance/coverage"
	"github.com/Clyra-AI/axym/core/compliance/framework"
	"github.com/Clyra-AI/axym/core/compliance/match"
	"github.com/Clyra-AI/axym/core/compliance/threshold"
	"github.com/Clyra-AI/axym/core/store"
)

var defaultFrameworkIDs = []string{"eu-ai-act", "soc2"}

type complianceRun struct {
	frameworks     []framework.Definition
	matchResult    match.Result
	coverageReport coverage.Report
	thresholdEval  threshold.Evaluation
}

func runCompliance(frameworkIDs []string, storeDir string, policyPath string, thresholdOverride *float64) (complianceRun, error) {
	definitions, err := framework.LoadMany(frameworkIDs)
	if err != nil {
		return complianceRun{}, err
	}
	st, err := store.New(store.Config{RootDir: storeDir})
	if err != nil {
		return complianceRun{}, fmt.Errorf("initialize local store: %w", err)
	}
	chain, err := st.LoadChain()
	if err != nil {
		return complianceRun{}, fmt.Errorf("load chain: %w", err)
	}

	matchResult := match.Evaluate(definitions, chain.Records, match.Options{ExcludeInvalidEvidence: true})
	coverageReport := coverage.Build(matchResult)
	policy, err := threshold.LoadPolicy(policyPath)
	if err != nil {
		return complianceRun{}, err
	}
	thresholdEval := threshold.Evaluate(toFrameworkCoverage(coverageReport), policy, thresholdOverride)
	return complianceRun{
		frameworks:     definitions,
		matchResult:    matchResult,
		coverageReport: coverageReport,
		thresholdEval:  thresholdEval,
	}, nil
}

func toFrameworkCoverage(report coverage.Report) []threshold.FrameworkCoverage {
	frameworks := make([]threshold.FrameworkCoverage, 0, len(report.Frameworks))
	for _, frameworkCoverage := range report.Frameworks {
		frameworks = append(frameworks, threshold.FrameworkCoverage{
			FrameworkID:     frameworkCoverage.FrameworkID,
			Coverage:        frameworkCoverage.Coverage,
			FailingControls: append([]string(nil), frameworkCoverage.FailingControl...),
		})
	}
	return frameworks
}

func normalizeFrameworkInput(values []string) []string {
	result := make([]string, 0)
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
	}
	if len(result) == 0 {
		return append([]string(nil), defaultFrameworkIDs...)
	}
	return result
}

func thresholdOverride(value float64) *float64 {
	if value < 0 {
		return nil
	}
	clamped := value
	if clamped < 0 {
		clamped = 0
	}
	if clamped > 1 {
		clamped = 1
	}
	return &clamped
}
