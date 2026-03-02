package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Clyra-AI/axym/core/compliance/framework"
	"github.com/Clyra-AI/axym/core/compliance/threshold"
	"github.com/spf13/cobra"
)

func newMapCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	var frameworks []string
	var storeDir string
	var policyConfig string
	var minCoverage float64

	cmd := &cobra.Command{
		Use:   "map",
		Short: "Map chain evidence to compliance controls",
		RunE: func(cmd *cobra.Command, args []string) error {
			normalizedFrameworks := normalizeFrameworkInput(frameworks)
			run, err := runCompliance(normalizedFrameworks, storeDir, policyConfig, thresholdOverride(minCoverage))
			if err != nil {
				return emitMapError(err, stdout, stderr, global)
			}

			data := map[string]any{
				"frameworks": run.matchResult.Frameworks,
				"summary":    run.matchResult.Summary,
				"coverage":   run.coverageReport,
				"threshold":  run.thresholdEval,
			}
			if !run.thresholdEval.Passed {
				if global.JSON {
					_ = printJSON(stdout, envelope{
						OK:      false,
						Command: "map",
						Data:    data,
						Error: &errorEnvelope{
							Reason:  threshold.ReasonThresholdNotMet,
							Message: "coverage threshold not met",
						},
					})
				}
				return &cliError{code: exitRegressionDrift, msg: "coverage threshold not met"}
			}

			if global.JSON {
				return printJSON(stdout, envelope{OK: true, Command: "map", Data: data})
			}
			printText(stdout, fmt.Sprintf("map: %d controls (%d covered, %d partial, %d gap)", run.matchResult.Summary.ControlCount, run.matchResult.Summary.CoveredCount, run.matchResult.Summary.PartialCount, run.matchResult.Summary.GapCount), global.Quiet)
			if global.Explain && !global.Quiet {
				for _, frameworkResult := range run.matchResult.Frameworks {
					for _, control := range frameworkResult.Controls {
						_, _ = fmt.Fprintln(stdout, fmt.Sprintf("%s/%s: %s", frameworkResult.ID, control.ControlID, control.Rationale))
					}
				}
			}
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&frameworks, "frameworks", nil, "Framework IDs to map (comma-separated), e.g. eu-ai-act,soc2")
	cmd.Flags().StringVar(&storeDir, "store-dir", ".axym", "Path to local chain store")
	cmd.Flags().StringVar(&policyConfig, "policy-config", "", "Optional JSON policy config with coverage thresholds")
	cmd.Flags().Float64Var(&minCoverage, "min-coverage", -1, "Optional minimum coverage threshold (0..1)")
	return cmd
}

func emitMapError(err error, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	var frameworkErr *framework.Error
	if errors.As(err, &frameworkErr) {
		if global.JSON {
			_ = printJSON(stdout, envelope{
				OK:      false,
				Command: "map",
				Error: &errorEnvelope{
					Reason:  strings.ToLower(frameworkErr.ReasonCode),
					Message: frameworkErr.Message,
				},
			})
		} else if !global.Quiet {
			_, _ = fmt.Fprintln(stderr, frameworkErr.Error())
		}
		return &cliError{code: exitInvalidInput, msg: frameworkErr.Error()}
	}

	if global.JSON {
		_ = printJSON(stdout, envelope{OK: false, Command: "map", Error: &errorEnvelope{Reason: "runtime_failure", Message: err.Error()}})
	} else if !global.Quiet {
		_, _ = fmt.Fprintln(stderr, err.Error())
	}
	return &cliError{code: exitRuntimeFailure, msg: err.Error()}
}
