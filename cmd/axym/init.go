package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/Clyra-AI/axym/core/config"
	"github.com/Clyra-AI/axym/core/samplepack"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/spf13/cobra"
)

func newInitCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	var storeDir string
	var policyPath string
	var samplePackDir string
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize deterministic Axym policy and local store",
		RunE: func(cmd *cobra.Command, args []string) error {
			var preparedSamplePack *samplepack.Prepared
			if strings.TrimSpace(samplePackDir) != "" {
				normalized, err := samplepack.ValidateTarget(samplePackDir)
				if err != nil {
					return emitInitInvalidInput(fmt.Sprintf("sample-pack initialization failed: %v", err), stdout, stderr, global)
				}
				samplePackDir = normalized
				preparedSamplePack, err = samplepack.Prepare(samplePackDir)
				if err != nil {
					return emitInitError(fmt.Errorf("prepare sample pack: %w", err), stdout, stderr, global)
				}
			}
			_, err := store.New(store.Config{RootDir: storeDir})
			if err != nil {
				if preparedSamplePack != nil {
					_ = preparedSamplePack.Cleanup()
				}
				return emitInitError(fmt.Errorf("initialize store: %w", err), stdout, stderr, global)
			}
			policy, created, err := config.WriteDefault(policyPath, force)
			if err != nil {
				if preparedSamplePack != nil {
					_ = preparedSamplePack.Cleanup()
				}
				return emitInitInvalidInput(fmt.Sprintf("policy initialization failed: %v", err), stdout, stderr, global)
			}
			data := map[string]any{
				"store_dir":      storeDir,
				"policy_path":    policy.PolicyPath,
				"policy_created": created,
				"version":        policy.Version,
			}
			if strings.TrimSpace(samplePackDir) != "" {
				pack, err := preparedSamplePack.Finalize()
				if err != nil {
					return emitInitError(fmt.Errorf("create sample pack: %w", err), stdout, stderr, global)
				}
				data["sample_pack"] = pack
			}
			if global.JSON {
				return printJSON(stdout, envelope{OK: true, Command: "init", Data: data})
			}
			message := fmt.Sprintf("initialized store=%s policy=%s", storeDir, policy.PolicyPath)
			if strings.TrimSpace(samplePackDir) != "" {
				message = fmt.Sprintf("%s sample_pack=%s", message, samplePackDir)
			}
			printText(stdout, message, global.Quiet)
			if global.Explain && !global.Quiet {
				printText(stdout, fmt.Sprintf("policy_created=%t version=%s", created, policy.Version), global.Quiet)
				if pack, ok := data["sample_pack"].(samplepack.Result); ok {
					for _, step := range pack.NextSteps {
						printText(stdout, step, global.Quiet)
					}
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&storeDir, "store-dir", ".axym", "Path to local chain store")
	cmd.Flags().StringVar(&policyPath, "policy-path", config.DefaultPolicyFile, "Path to Axym policy file")
	cmd.Flags().StringVar(&samplePackDir, "sample-pack", "", "Materialize a deterministic local sample pack at the provided path")
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
