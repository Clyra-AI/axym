package main

import (
	"errors"
	"fmt"
	"io"
	"time"

	coreCollect "github.com/Clyra-AI/axym/core/collect"
	"github.com/Clyra-AI/axym/core/collector"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/redact"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/spf13/cobra"
)

func newCollectCmd(stdout io.Writer, global *globalFlags) *cobra.Command {
	var dryRun bool
	var storeDir string
	var fixtureDir string
	var sinkMode string
	var pluginCommands []string
	var pluginTimeout time.Duration
	var governanceEventFiles []string

	cmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect evidence",
		RunE: func(cmd *cobra.Command, args []string) error {
			request := collector.Request{
				Now:                  time.Now().UTC().Truncate(time.Second),
				FixtureDir:           fixtureDir,
				PluginCommands:       pluginCommands,
				PluginTimeout:        pluginTimeout,
				GovernanceEventFiles: governanceEventFiles,
			}
			registry, err := coreCollect.BuildRegistry(request)
			if err != nil {
				return emitCollectError(&coreCollect.Error{
					ReasonCode: coreCollect.ReasonRuntime,
					Message:    "build collector registry",
					ExitCode:   exitRuntimeFailure,
					Err:        err,
				}, stdout, global)
			}

			var evidenceStore *store.Store
			if !dryRun {
				evidenceStore, err = store.New(store.Config{
					RootDir:        storeDir,
					ComplianceMode: sink.NormalizeMode(sink.Mode(sinkMode)) == sink.ModeFailClosed,
				})
				if err != nil {
					return emitCollectError(&coreCollect.Error{
						ReasonCode: "SINK_UNAVAILABLE",
						Message:    "initialize local store",
						ExitCode:   exitRuntimeFailure,
						Err:        err,
					}, stdout, global)
				}
			}

			runner := coreCollect.Runner{
				Registry: registry,
				Store:    evidenceStore,
				SinkMode: sink.Mode(sinkMode),
				Redaction: redact.Config{
					EventRules: []redact.Rule{
						{Path: "api_key", Action: redact.ActionHash},
						{Path: "token", Action: redact.ActionHash},
						{Path: "authorization", Action: redact.ActionHash},
						{Path: "password", Action: redact.ActionHash},
						{Path: "secret", Action: redact.ActionHash},
						{Path: "prompt", Action: redact.ActionHash},
						{Path: "instruction_text", Action: redact.ActionHash},
						{Path: "system_prompt", Action: redact.ActionHash},
						{Path: "knowledge_body", Action: redact.ActionHash},
						{Path: "knowledge_text", Action: redact.ActionHash},
						{Path: "context_snapshot", Action: redact.ActionHash},
						{Path: "query_text", Action: redact.ActionHash},
						{Path: "sql", Action: redact.ActionHash},
					},
					MetadataRules: []redact.Rule{
						{Path: "api_key", Action: redact.ActionHash},
						{Path: "auth_token", Action: redact.ActionHash},
						{Path: "token", Action: redact.ActionHash},
						{Path: "authorization", Action: redact.ActionHash},
						{Path: "password", Action: redact.ActionHash},
						{Path: "secret", Action: redact.ActionHash},
						{Path: "prompt", Action: redact.ActionHash},
						{Path: "instruction_text", Action: redact.ActionHash},
						{Path: "system_prompt", Action: redact.ActionHash},
						{Path: "knowledge_body", Action: redact.ActionHash},
						{Path: "knowledge_text", Action: redact.ActionHash},
						{Path: "context_snapshot", Action: redact.ActionHash},
						{Path: "signature", Action: redact.ActionHash},
					},
				},
			}
			result, err := runner.Run(cmd.Context(), request, dryRun)
			if err != nil {
				return emitCollectError(err, stdout, global)
			}
			if global.JSON {
				return printJSON(stdout, envelope{OK: true, Command: "collect", Data: result})
			}
			if !global.Quiet {
				printText(stdout, fmt.Sprintf("collect %s: %d captured, %d rejected, %d failures", modeLabel(dryRun), result.Captured, result.Rejected, result.Failures), global.Quiet)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate collection configuration without writing artifacts")
	cmd.Flags().StringVar(&storeDir, "store-dir", ".axym", "Path to local chain store")
	cmd.Flags().StringVar(&fixtureDir, "fixture-dir", "", "Optional fixture directory for deterministic test inputs")
	cmd.Flags().StringVar(&sinkMode, "sink-mode", "fail_closed", "Sink policy mode: fail_closed|advisory_only|shadow")
	cmd.Flags().StringSliceVar(&pluginCommands, "plugin", nil, "Plugin command(s) implementing collector protocol")
	cmd.Flags().DurationVar(&pluginTimeout, "plugin-timeout", 2*time.Second, "Timeout for each plugin collector")
	cmd.Flags().StringSliceVar(&governanceEventFiles, "governance-event-file", nil, "JSONL governance event file(s) to promote")
	return cmd
}

func modeLabel(dryRun bool) string {
	if dryRun {
		return "dry-run"
	}
	return "write"
}

func emitCollectError(err error, stdout io.Writer, global *globalFlags) error {
	var collectErr *coreCollect.Error
	if errors.As(err, &collectErr) {
		if global.JSON {
			_ = printJSON(stdout, envelope{
				OK:      false,
				Command: "collect",
				Error: &errorEnvelope{
					Reason:  collectErr.ReasonCode,
					Message: collectErr.Message,
				},
			})
		}
		return &cliError{code: collectErr.ExitCode, msg: collectErr.Error()}
	}
	if global.JSON {
		_ = printJSON(stdout, envelope{
			OK:      false,
			Command: "collect",
			Error:   &errorEnvelope{Reason: coreCollect.ReasonRuntime, Message: err.Error()},
		})
	}
	return &cliError{code: exitRuntimeFailure, msg: err.Error()}
}
