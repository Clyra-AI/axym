package main

import (
	"errors"
	"fmt"
	"io"

	corereplay "github.com/Clyra-AI/axym/core/replay"
	"github.com/spf13/cobra"
)

func newReplayCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	var model string
	var tier string
	var storeDir string
	var productionCritical bool
	var dataSensitivity string
	var publicExposure bool

	cmd := &cobra.Command{
		Use:   "replay",
		Short: "Emit deterministic replay certification evidence",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := corereplay.Run(corereplay.Request{
				Model:    model,
				Tier:     tier,
				StoreDir: storeDir,
				Risk: corereplay.RiskInput{
					ProductionCritical: productionCritical,
					DataSensitivity:    dataSensitivity,
					PublicExposure:     publicExposure,
				},
			})
			if err != nil {
				return emitReplayError(err, stdout, stderr, global)
			}
			if global.JSON {
				return printJSON(stdout, envelope{OK: true, Command: "replay", Data: result})
			}
			printText(stdout, fmt.Sprintf("replay certified: %s tier=%s", result.Model, result.Tier), global.Quiet)
			if global.Explain && !global.Quiet {
				_, _ = fmt.Fprintf(stdout, "blast_radius=%v\n", result.BlastRadius)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&model, "model", "", "Model identifier for replay run")
	cmd.Flags().StringVar(&tier, "tier", "", "Replay tier (A|B|C); omitted => deterministic classification from risk flags")
	cmd.Flags().StringVar(&storeDir, "store-dir", ".axym", "Path to local chain store")
	cmd.Flags().BoolVar(&productionCritical, "production-critical", false, "Mark replay target as production critical")
	cmd.Flags().StringVar(&dataSensitivity, "data-sensitivity", "", "Data sensitivity (high|medium|low)")
	cmd.Flags().BoolVar(&publicExposure, "public-exposure", false, "Mark replay target as externally exposed")
	return cmd
}

func emitReplayError(err error, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	var replayErr *corereplay.Error
	if errors.As(err, &replayErr) {
		if global.JSON {
			_ = printJSON(stdout, envelope{
				OK:      false,
				Command: "replay",
				Error: &errorEnvelope{
					Reason:  replayErr.ReasonCode,
					Message: replayErr.Message,
				},
			})
		} else if !global.Quiet {
			_, _ = fmt.Fprintln(stderr, replayErr.Error())
		}
		code := exitRuntimeFailure
		if replayErr.ReasonCode == corereplay.ReasonInvalidInput {
			code = exitInvalidInput
		}
		return &cliError{code: code, msg: replayErr.Error()}
	}
	if global.JSON {
		_ = printJSON(stdout, envelope{OK: false, Command: "replay", Error: &errorEnvelope{Reason: "runtime_failure", Message: err.Error()}})
	}
	return &cliError{code: exitRuntimeFailure, msg: err.Error()}
}
