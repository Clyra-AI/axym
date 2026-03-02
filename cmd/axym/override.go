package main

import (
	"errors"
	"fmt"
	"io"
	"time"

	coreoverride "github.com/Clyra-AI/axym/core/override"
	"github.com/spf13/cobra"
)

func newOverrideCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "override",
		Short: "Create signed override artifacts",
	}
	cmd.AddCommand(newOverrideCreateCmd(stdout, stderr, global))
	return cmd
}

func newOverrideCreateCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	var bundle string
	var reason string
	var signer string
	var expires string
	var storeDir string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a signed append-only override",
		RunE: func(cmd *cobra.Command, args []string) error {
			var expiresAt time.Time
			if expires != "" {
				parsed, err := time.Parse(time.RFC3339, expires)
				if err != nil {
					return emitOverrideInvalidInput("expires must use RFC3339", stdout, stderr, global)
				}
				expiresAt = parsed
			}
			result, err := coreoverride.Create(coreoverride.Request{
				Bundle:    bundle,
				Reason:    reason,
				Signer:    signer,
				StoreDir:  storeDir,
				ExpiresAt: expiresAt,
			})
			if err != nil {
				return emitOverrideError(err, stdout, stderr, global)
			}
			if global.JSON {
				return printJSON(stdout, envelope{OK: true, Command: "override", Data: result})
			}
			printText(stdout, fmt.Sprintf("override created: %s", result.RecordID), global.Quiet)
			if global.Explain && !global.Quiet {
				_, _ = fmt.Fprintf(stdout, "bundle=%s signer=%s expires_at=%s artifact=%s\n", result.Bundle, result.Signer, result.ExpiresAt, result.ArtifactPath)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&bundle, "bundle", "", "Bundle identifier for override scope")
	cmd.Flags().StringVar(&reason, "reason", "", "Override rationale")
	cmd.Flags().StringVar(&signer, "signer", "", "Signer identity")
	cmd.Flags().StringVar(&expires, "expires", "", "Optional override expiry timestamp (RFC3339)")
	cmd.Flags().StringVar(&storeDir, "store-dir", ".axym", "Path to local chain store")
	return cmd
}

func emitOverrideError(err error, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	var overrideErr *coreoverride.Error
	if errors.As(err, &overrideErr) {
		if global.JSON {
			_ = printJSON(stdout, envelope{
				OK:      false,
				Command: "override",
				Error: &errorEnvelope{
					Reason:  overrideErr.ReasonCode,
					Message: overrideErr.Message,
				},
			})
		} else if !global.Quiet {
			_, _ = fmt.Fprintln(stderr, overrideErr.Error())
		}
		code := exitRuntimeFailure
		if overrideErr.ReasonCode == coreoverride.ReasonInvalidInput {
			code = exitInvalidInput
		}
		return &cliError{code: code, msg: overrideErr.Error()}
	}
	if global.JSON {
		_ = printJSON(stdout, envelope{OK: false, Command: "override", Error: &errorEnvelope{Reason: "runtime_failure", Message: err.Error()}})
	}
	return &cliError{code: exitRuntimeFailure, msg: err.Error()}
}

func emitOverrideInvalidInput(message string, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	if global.JSON {
		_ = printJSON(stdout, envelope{OK: false, Command: "override", Error: &errorEnvelope{Reason: "invalid_input", Message: message}})
	} else if !global.Quiet {
		_, _ = fmt.Fprintln(stderr, message)
	}
	return &cliError{code: exitInvalidInput, msg: message}
}
