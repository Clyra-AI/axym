package main

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"

	coreVerify "github.com/Clyra-AI/axym/core/verify"
	"github.com/spf13/cobra"
)

type cliError struct {
	code int
	msg  string
}

func (e *cliError) Error() string { return e.msg }
func (e *cliError) ExitCode() int { return e.code }

func newVerifyCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	var chain bool
	var bundlePath string
	var frameworks []string
	var storeDir string
	var tempDir string

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify chain or bundle integrity",
		RunE: func(cmd *cobra.Command, args []string) error {
			if (chain && bundlePath != "") || (!chain && bundlePath == "") {
				return emitVerifyInvalidInput("specify exactly one of --chain or --bundle", stdout, stderr, global)
			}
			if chain {
				result, err := coreVerify.VerifyChainFromStoreDir(storeDir)
				if err != nil {
					return emitVerifyError(err, stdout, stderr, global)
				}
				if global.JSON {
					return printJSON(stdout, envelope{OK: true, Command: "verify", Data: map[string]any{"target": "chain", "verification": result}})
				}
				printText(stdout, fmt.Sprintf("Chain intact. %d records. No gaps.", result.Count), global.Quiet)
				return nil
			}

			if tempDir == "" {
				tempDir = filepath.Join(storeDir, "tmp", "verify")
			}
			if err := coreVerify.EnsureManagedTempDir(tempDir); err != nil {
				return emitVerifyError(err, stdout, stderr, global)
			}
			result, err := coreVerify.VerifyBundle(bundlePath, normalizeFrameworkInput(frameworks))
			if err != nil {
				return emitVerifyError(err, stdout, stderr, global)
			}
			if global.JSON {
				return printJSON(stdout, envelope{OK: true, Command: "verify", Data: map[string]any{"target": "bundle", "verification": result}})
			}
			printText(stdout, fmt.Sprintf("Bundle verified: %s (%d files)", result.Path, result.Files), global.Quiet)
			return nil
		},
	}

	cmd.Flags().BoolVar(&chain, "chain", false, "Verify local append-only chain")
	cmd.Flags().StringVar(&bundlePath, "bundle", "", "Verify bundle directory")
	cmd.Flags().StringSliceVar(&frameworks, "frameworks", nil, "Framework IDs for compliance completeness checks (comma-separated)")
	cmd.Flags().StringVar(&storeDir, "store-dir", ".axym", "Path to local chain store")
	cmd.Flags().StringVar(&tempDir, "temp-dir", "", "Path for verify temporary artifacts")
	return cmd
}

func emitVerifyError(err error, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	var vErr *coreVerify.Error
	if errors.As(err, &vErr) {
		if global.JSON {
			_ = printJSON(stdout, envelope{
				OK:      false,
				Command: "verify",
				Error: &errorEnvelope{
					Reason:     vErr.ReasonCode,
					Message:    vErr.Message,
					BreakIndex: vErr.BreakIndex,
					BreakPoint: vErr.BreakPoint,
				},
			})
		} else if !global.Quiet {
			_, _ = fmt.Fprintln(stderr, vErr.Error())
		}
		return &cliError{code: vErr.ExitCode, msg: vErr.Error()}
	}

	if global.JSON {
		_ = printJSON(stdout, envelope{
			OK:      false,
			Command: "verify",
			Error:   &errorEnvelope{Reason: "runtime_failure", Message: err.Error()},
		})
	}
	return &cliError{code: exitRuntimeFailure, msg: err.Error()}
}

func emitVerifyInvalidInput(message string, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	if global.JSON {
		_ = printJSON(stdout, envelope{
			OK:      false,
			Command: "verify",
			Error:   &errorEnvelope{Reason: "invalid_input", Message: message},
		})
	} else if !global.Quiet {
		_, _ = fmt.Fprintln(stderr, message)
	}
	return &cliError{code: exitInvalidInput, msg: message}
}
