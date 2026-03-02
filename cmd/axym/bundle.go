package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	corebundle "github.com/Clyra-AI/axym/core/bundle"
	"github.com/spf13/cobra"
)

func newBundleCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	var auditName string
	var frameworks []string
	var storeDir string
	var outputDir string

	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Assemble deterministic signed audit bundle",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(auditName) == "" {
				return emitBundleInvalidInput("audit name is required", stdout, stderr, global)
			}
			result, err := corebundle.Build(corebundle.BuildRequest{
				AuditName:    auditName,
				FrameworkIDs: normalizeFrameworkInput(frameworks),
				StoreDir:     storeDir,
				OutputDir:    outputDir,
			})
			if err != nil {
				return emitBundleError(err, stdout, stderr, global)
			}

			data := map[string]any{
				"path":         result.Path,
				"files":        result.Files,
				"algo":         result.Algo,
				"audit":        result.AuditName,
				"frameworks":   result.Frameworks,
				"verification": result.Verification,
				"compliance":   result.Compliance,
			}
			if global.JSON {
				return printJSON(stdout, envelope{OK: true, Command: "bundle", Data: data})
			}
			printText(stdout, fmt.Sprintf("bundle: %s (%d files)", result.Path, result.Files), global.Quiet)
			if global.Explain && !global.Quiet {
				_, _ = fmt.Fprintf(stdout, "frameworks=%s compliance_complete=%t\n", strings.Join(result.Frameworks, ","), result.Compliance.Complete)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&auditName, "audit", "", "Audit run name for bundle metadata")
	cmd.Flags().StringSliceVar(&frameworks, "frameworks", nil, "Framework IDs (comma-separated), e.g. eu-ai-act,soc2")
	cmd.Flags().StringVar(&storeDir, "store-dir", ".axym", "Path to local chain store")
	cmd.Flags().StringVar(&outputDir, "output", corebundle.DefaultOutputDir, "Path to bundle output directory")
	return cmd
}

func emitBundleError(err error, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	var bundleErr *corebundle.Error
	if errors.As(err, &bundleErr) {
		if global.JSON {
			_ = printJSON(stdout, envelope{
				OK:      false,
				Command: "bundle",
				Error: &errorEnvelope{
					Reason:  bundleErr.ReasonCode,
					Message: bundleErr.Message,
				},
			})
		} else if !global.Quiet {
			_, _ = fmt.Fprintln(stderr, bundleErr.Error())
		}
		return &cliError{code: bundleErr.ExitCode, msg: bundleErr.Error()}
	}

	if global.JSON {
		_ = printJSON(stdout, envelope{
			OK:      false,
			Command: "bundle",
			Error:   &errorEnvelope{Reason: "runtime_failure", Message: err.Error()},
		})
	}
	return &cliError{code: exitRuntimeFailure, msg: err.Error()}
}

func emitBundleInvalidInput(message string, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	if global.JSON {
		_ = printJSON(stdout, envelope{
			OK:      false,
			Command: "bundle",
			Error:   &errorEnvelope{Reason: "invalid_input", Message: message},
		})
	} else if !global.Quiet {
		_, _ = fmt.Fprintln(stderr, message)
	}
	return &cliError{code: exitInvalidInput, msg: message}
}
