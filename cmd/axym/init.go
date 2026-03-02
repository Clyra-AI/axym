package main

import (
	"fmt"
	"io"

	"github.com/Clyra-AI/axym/core/config"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/spf13/cobra"
)

func newInitCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	var storeDir string
	var policyPath string
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize deterministic Axym policy and local store",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := store.New(store.Config{RootDir: storeDir})
			if err != nil {
				return emitInitError(fmt.Errorf("initialize store: %w", err), stdout, stderr, global)
			}
			policy, created, err := config.WriteDefault(policyPath, force)
			if err != nil {
				return emitInitInvalidInput(fmt.Sprintf("policy initialization failed: %v", err), stdout, stderr, global)
			}
			data := map[string]any{
				"store_dir":      storeDir,
				"policy_path":    policy.PolicyPath,
				"policy_created": created,
				"version":        policy.Version,
			}
			if global.JSON {
				return printJSON(stdout, envelope{OK: true, Command: "init", Data: data})
			}
			printText(stdout, fmt.Sprintf("initialized store=%s policy=%s", storeDir, policy.PolicyPath), global.Quiet)
			if global.Explain && !global.Quiet {
				printText(stdout, fmt.Sprintf("policy_created=%t version=%s", created, policy.Version), global.Quiet)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&storeDir, "store-dir", ".axym", "Path to local chain store")
	cmd.Flags().StringVar(&policyPath, "policy-path", config.DefaultPolicyFile, "Path to Axym policy file")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing policy file with defaults")
	return cmd
}

func emitInitError(err error, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	if global.JSON {
		_ = printJSON(stdout, envelope{
			OK:      false,
			Command: "init",
			Error: &errorEnvelope{
				Reason:  "runtime_failure",
				Message: err.Error(),
			},
		})
	} else if !global.Quiet {
		_, _ = fmt.Fprintln(stderr, err.Error())
	}
	return &cliError{code: exitRuntimeFailure, msg: err.Error()}
}

func emitInitInvalidInput(message string, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	if global.JSON {
		_ = printJSON(stdout, envelope{
			OK:      false,
			Command: "init",
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
