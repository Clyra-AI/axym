package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Clyra-AI/axym/core/compliance/framework"
	"github.com/Clyra-AI/axym/core/compliance/threshold"
	"github.com/Clyra-AI/axym/core/gaps"
	"github.com/spf13/cobra"
)

func newGapsCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	var frameworks []string
	var storeDir string
	var policyConfig string
	var minCoverage float64

	cmd := &cobra.Command{
		Use:   "gaps",
		Short: "Compute deterministic compliance gaps",
		RunE: func(cmd *cobra.Command, args []string) error {
			policy, _, err := resolvePolicyRuntime(policyConfig)
			if err != nil {
				return emitGapsInvalidInput(fmt.Sprintf("invalid policy config: %v", err), stdout, stderr, global)
			}
			normalizedFrameworks := resolveFrameworkInput(policy, frameworks)
			explicitStoreDir := ""
			if cmd.Flags().Changed("store-dir") {
				explicitStoreDir = storeDir
			}
			resolvedStoreDir := resolveStoreDir(policy, explicitStoreDir)
			coverageOverride := thresholdOverride(minCoverage)
			if cmd.Flags().Changed("min-coverage") {
				if minCoverage < 0 || minCoverage > 1 {
					return emitGapsInvalidInput("min-coverage must be within [0,1]", stdout, stderr, global)
				}
				coverageOverride = &minCoverage
			}

			run, err := runCompliance(normalizedFrameworks, resolvedStoreDir, policy.ThresholdPolicy(), coverageOverride)
			if err != nil {
				return emitGapsError(err, stdout, stderr, global)
			}
			report := gaps.Build(run.coverageReport)
			data := map[string]any{
				"frameworks": run.coverageReport.Frameworks,
				"summary":    report.Summary,
				"gaps":       report.Gaps,
				"grade":      report.Grade,
				"threshold":  run.thresholdEval,
			}

			if !run.thresholdEval.Passed {
				if global.JSON {
					_ = printJSON(stdout, envelope{OK: false, Command: "gaps", Data: data, Error: &errorEnvelope{Reason: threshold.ReasonThresholdNotMet, Message: "coverage threshold not met"}})
				}
				return &cliError{code: exitRegressionDrift, msg: "coverage threshold not met"}
			}

			if global.JSON {
				return printJSON(stdout, envelope{OK: true, Command: "gaps", Data: data})
			}
			printText(stdout, fmt.Sprintf("gaps: %d items (grade %s)", len(report.Gaps), report.Grade.Letter), global.Quiet)
			if global.Explain && !global.Quiet {
				for _, line := range gaps.Explain(report) {
					_, _ = fmt.Fprintln(stdout, line)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&frameworks, "frameworks", nil, "Framework IDs to evaluate (comma-separated), e.g. eu-ai-act,soc2")
	cmd.Flags().StringVar(&storeDir, "store-dir", ".axym", "Path to local chain store")
	cmd.Flags().StringVar(&policyConfig, "policy-config", "", "Optional policy config path (defaults to axym-policy.yaml when present)")
	cmd.Flags().Float64Var(&minCoverage, "min-coverage", -1, "Optional minimum coverage threshold (0..1)")
	return cmd
}

func emitGapsError(err error, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	var frameworkErr *framework.Error
	if errors.As(err, &frameworkErr) {
		if global.JSON {
			_ = printJSON(stdout, envelope{OK: false, Command: "gaps", Error: &errorEnvelope{Reason: strings.ToLower(frameworkErr.ReasonCode), Message: frameworkErr.Message}})
		} else if !global.Quiet {
			_, _ = fmt.Fprintln(stderr, frameworkErr.Error())
		}
		return &cliError{code: exitInvalidInput, msg: frameworkErr.Error()}
	}

	if global.JSON {
		_ = printJSON(stdout, envelope{OK: false, Command: "gaps", Error: &errorEnvelope{Reason: "runtime_failure", Message: err.Error()}})
	} else if !global.Quiet {
		_, _ = fmt.Fprintln(stderr, err.Error())
	}
	return &cliError{code: exitRuntimeFailure, msg: err.Error()}
}

func emitGapsInvalidInput(message string, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	if global.JSON {
		_ = printJSON(stdout, envelope{OK: false, Command: "gaps", Error: &errorEnvelope{Reason: "invalid_input", Message: message}})
	} else if !global.Quiet {
		_, _ = fmt.Fprintln(stderr, message)
	}
	return &cliError{code: exitInvalidInput, msg: message}
}
