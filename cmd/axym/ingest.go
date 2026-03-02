package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/ingest/gait"
	"github.com/Clyra-AI/axym/core/ingest/stitch"
	"github.com/Clyra-AI/axym/core/ingest/wrkr"
	"github.com/Clyra-AI/axym/core/review/sessiongap"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/spf13/cobra"
)

const reasonIngestChainReadFailed = "INGEST_CHAIN_READ_FAILED"

func newIngestCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	var source string
	var inputPaths []string
	var storeDir string
	var stateDir string
	var sessionGapThreshold time.Duration

	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Ingest sibling proof artifacts (Wrkr, Gait)",
		RunE: func(cmd *cobra.Command, args []string) error {
			selectedSource := strings.ToLower(strings.TrimSpace(source))
			if selectedSource == "" {
				return emitIngestInvalidInput("source is required (wrkr|gait)", stdout, stderr, global)
			}

			evidenceStore, err := store.New(store.Config{RootDir: storeDir})
			if err != nil {
				return emitIngestError(&wrkr.Error{
					ReasonCode: wrkr.ReasonStateUnavailable,
					Message:    "initialize local store",
					Err:        err,
				}, stdout, stderr, global)
			}

			switch selectedSource {
			case "wrkr":
				result, err := wrkr.Ingest(context.Background(), wrkr.Request{
					InputPaths: inputPaths,
					Store:      evidenceStore,
					StateDir:   stateDir,
				})
				if err != nil {
					return emitIngestError(err, stdout, stderr, global)
				}
				stitching, stitchErr := buildStitchSummary(evidenceStore, sessionGapThreshold)
				if stitchErr != nil {
					return emitIngestError(stitchErr, stdout, stderr, global)
				}
				if global.JSON {
					return printJSON(stdout, envelope{
						OK:      true,
						Command: "ingest",
						Data: map[string]any{
							"source": selectedSource,
							"result": result,
							"stitch": stitching,
						},
					})
				}
				if !global.Quiet {
					printText(stdout, fmt.Sprintf("ingest %s: %d appended, %d deduped, %d rejected", selectedSource, result.Appended, result.Deduped, result.Rejected), global.Quiet)
				}
				return nil
			case "gait":
				result, err := gait.Ingest(context.Background(), gait.Request{
					InputPaths: inputPaths,
					Store:      evidenceStore,
				})
				if err != nil {
					return emitIngestError(err, stdout, stderr, global)
				}
				stitching, stitchErr := buildStitchSummary(evidenceStore, sessionGapThreshold)
				if stitchErr != nil {
					return emitIngestError(stitchErr, stdout, stderr, global)
				}
				if global.JSON {
					return printJSON(stdout, envelope{
						OK:      true,
						Command: "ingest",
						Data: map[string]any{
							"source": selectedSource,
							"result": result,
							"stitch": stitching,
						},
					})
				}
				if !global.Quiet {
					printText(stdout, fmt.Sprintf("ingest %s: %d appended, %d deduped, %d rejected", selectedSource, result.Appended, result.Deduped, result.Rejected), global.Quiet)
				}
				return nil
			default:
				return emitIngestInvalidInput("source must be one of: wrkr, gait", stdout, stderr, global)
			}
		},
	}

	cmd.Flags().StringVar(&source, "source", "", "Sibling source to ingest (wrkr|gait)")
	cmd.Flags().StringSliceVar(&inputPaths, "input", nil, "Input file or directory path(s)")
	cmd.Flags().StringVar(&storeDir, "store-dir", ".axym", "Path to local chain store")
	cmd.Flags().StringVar(&stateDir, "state-dir", ".axym", "Path to ingest state directory")
	cmd.Flags().DurationVar(&sessionGapThreshold, "session-gap-threshold", 30*time.Minute, "Maximum allowed time between adjacent records before signaling CHAIN_SESSION_GAP")
	return cmd
}

func buildStitchSummary(st *store.Store, threshold time.Duration) (map[string]any, error) {
	chain, err := st.LoadChain()
	if err != nil {
		return nil, &wrkr.Error{
			ReasonCode: reasonIngestChainReadFailed,
			Message:    "load chain for session stitching",
			Err:        err,
		}
	}
	stitchResult := stitch.Analyze(chain.Records, stitch.Config{MaxGap: threshold})
	signals := sessiongap.BuildSignals(stitchResult.Gaps)
	return map[string]any{
		"intact":         stitchResult.Intact,
		"gaps":           stitchResult.Gaps,
		"review_signals": signals,
	}, nil
}

func emitIngestError(err error, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	var ingestErr *wrkr.Error
	if errors.As(err, &ingestErr) {
		if global.JSON {
			_ = printJSON(stdout, envelope{
				OK:      false,
				Command: "ingest",
				Error: &errorEnvelope{
					Reason:  ingestErr.ReasonCode,
					Message: ingestErr.Message,
				},
			})
		} else if !global.Quiet {
			_, _ = fmt.Fprintln(stderr, ingestErr.Error())
		}
		code := exitRuntimeFailure
		if ingestErr.ReasonCode == wrkr.ReasonInvalidInput {
			code = exitInvalidInput
		}
		return &cliError{code: code, msg: ingestErr.Error()}
	}
	var gaitErr *gait.Error
	if errors.As(err, &gaitErr) {
		if global.JSON {
			_ = printJSON(stdout, envelope{
				OK:      false,
				Command: "ingest",
				Error: &errorEnvelope{
					Reason:  gaitErr.ReasonCode,
					Message: gaitErr.Message,
				},
			})
		} else if !global.Quiet {
			_, _ = fmt.Fprintln(stderr, gaitErr.Error())
		}
		code := exitRuntimeFailure
		if gaitErr.ReasonCode == gait.ReasonInvalidInput {
			code = exitInvalidInput
		}
		return &cliError{code: code, msg: gaitErr.Error()}
	}

	if global.JSON {
		_ = printJSON(stdout, envelope{
			OK:      false,
			Command: "ingest",
			Error: &errorEnvelope{
				Reason:  "runtime_failure",
				Message: err.Error(),
			},
		})
	}
	return &cliError{code: exitRuntimeFailure, msg: err.Error()}
}

func emitIngestInvalidInput(message string, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	if global.JSON {
		_ = printJSON(stdout, envelope{
			OK:      false,
			Command: "ingest",
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
