package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/compliance/framework"
	"github.com/Clyra-AI/axym/core/regress"
	"github.com/spf13/cobra"
)

func newRegressCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "regress",
		Short: "Capture and run deterministic compliance regression baselines",
	}
	cmd.AddCommand(newRegressInitCmd(stdout, stderr, global))
	cmd.AddCommand(newRegressRunCmd(stdout, stderr, global))
	return cmd
}

func newRegressInitCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	var baselinePath string
	var frameworks []string
	var storeDir string
	var policyConfig string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Capture a regression baseline from current coverage",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(baselinePath) == "" {
				return emitRegressInvalidInput("baseline path is required", "regress", stdout, stderr, global)
			}
			policy, _, err := resolvePolicyRuntime(policyConfig)
			if err != nil {
				return emitRegressInvalidInput(fmt.Sprintf("invalid policy config: %v", err), "regress", stdout, stderr, global)
			}
			explicitStoreDir := ""
			if cmd.Flags().Changed("store-dir") {
				explicitStoreDir = storeDir
			}
			run, err := runCompliance(resolveFrameworkInput(policy, frameworks), resolveStoreDir(policy, explicitStoreDir), policy.ThresholdPolicy(), nil)
			if err != nil {
				return emitRegressError(err, "regress", stdout, stderr, global)
			}
			result, err := regress.Init(regress.InitRequest{
				BaselinePath:   baselinePath,
				CoverageReport: run.coverageReport,
				Now:            time.Now().UTC().Truncate(time.Second),
			})
			if err != nil {
				return emitRegressError(err, "regress", stdout, stderr, global)
			}
			if global.JSON {
				return printJSON(stdout, envelope{OK: true, Command: "regress", Data: map[string]any{"action": "init", "result": result}})
			}
			printText(stdout, fmt.Sprintf("regress init: %s (%d frameworks)", result.Path, result.Frameworks), global.Quiet)
			return nil
		},
	}
	cmd.Flags().StringVar(&baselinePath, "baseline", "", "Path to baseline file or directory")
	cmd.Flags().StringSliceVar(&frameworks, "frameworks", nil, "Framework IDs (comma-separated), e.g. eu-ai-act,soc2")
	cmd.Flags().StringVar(&storeDir, "store-dir", ".axym", "Path to local chain store")
	cmd.Flags().StringVar(&policyConfig, "policy-config", "", "Optional policy config path (defaults to axym-policy.yaml when present)")
	return cmd
}

func newRegressRunCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	var baselinePath string
	var frameworks []string
	var storeDir string
	var policyConfig string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run drift detection against a regression baseline",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(baselinePath) == "" {
				return emitRegressInvalidInput("baseline path is required", "regress", stdout, stderr, global)
			}
			policy, _, err := resolvePolicyRuntime(policyConfig)
			if err != nil {
				return emitRegressInvalidInput(fmt.Sprintf("invalid policy config: %v", err), "regress", stdout, stderr, global)
			}
			explicitStoreDir := ""
			if cmd.Flags().Changed("store-dir") {
				explicitStoreDir = storeDir
			}
			run, err := runCompliance(resolveFrameworkInput(policy, frameworks), resolveStoreDir(policy, explicitStoreDir), policy.ThresholdPolicy(), nil)
			if err != nil {
				return emitRegressError(err, "regress", stdout, stderr, global)
			}
			result, err := regress.Run(regress.RunRequest{
				BaselinePath:   baselinePath,
				CoverageReport: run.coverageReport,
			})
			if err != nil {
				return emitRegressError(err, "regress", stdout, stderr, global)
			}
			if result.DriftDetected {
				if global.JSON {
					_ = printJSON(stdout, envelope{
						OK:      false,
						Command: "regress",
						Data:    map[string]any{"action": "run", "result": result},
						Error: &errorEnvelope{
							Reason:  "regression_drift",
							Message: "coverage drift detected",
						},
					})
				}
				return &cliError{code: exitRegressionDrift, msg: "coverage drift detected"}
			}
			if global.JSON {
				return printJSON(stdout, envelope{OK: true, Command: "regress", Data: map[string]any{"action": "run", "result": result}})
			}
			printText(stdout, fmt.Sprintf("regress run: no drift (%s)", result.BaselinePath), global.Quiet)
			return nil
		},
	}
	cmd.Flags().StringVar(&baselinePath, "baseline", "", "Path to baseline file or directory")
	cmd.Flags().StringSliceVar(&frameworks, "frameworks", nil, "Framework IDs (comma-separated), e.g. eu-ai-act,soc2")
	cmd.Flags().StringVar(&storeDir, "store-dir", ".axym", "Path to local chain store")
	cmd.Flags().StringVar(&policyConfig, "policy-config", "", "Optional policy config path (defaults to axym-policy.yaml when present)")
	return cmd
}

func emitRegressError(err error, command string, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	var frameworkErr *framework.Error
	if errors.As(err, &frameworkErr) {
		if global.JSON {
			_ = printJSON(stdout, envelope{
				OK:      false,
				Command: command,
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

	var regressErr *regress.Error
	if errors.As(err, &regressErr) {
		if global.JSON {
			_ = printJSON(stdout, envelope{
				OK:      false,
				Command: command,
				Error: &errorEnvelope{
					Reason:  regressErr.ReasonCode,
					Message: regressErr.Message,
				},
			})
		} else if !global.Quiet {
			_, _ = fmt.Fprintln(stderr, regressErr.Error())
		}
		return &cliError{code: regressErr.ExitCode, msg: regressErr.Error()}
	}
	if global.JSON {
		_ = printJSON(stdout, envelope{
			OK:      false,
			Command: command,
			Error: &errorEnvelope{
				Reason:  "runtime_failure",
				Message: err.Error(),
			},
		})
	}
	return &cliError{code: exitRuntimeFailure, msg: err.Error()}
}

func emitRegressInvalidInput(message string, command string, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	if global.JSON {
		_ = printJSON(stdout, envelope{
			OK:      false,
			Command: command,
			Error: &errorEnvelope{
				Reason:  "invalid_input",
				Message: message,
			},
		})
	} else if !global.Quiet {
		_, _ = fmt.Fprintln(stderr, message)
	}
	return &cliError{code: exitInvalidInput, msg: message}
}
