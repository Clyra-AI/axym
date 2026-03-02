package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	corereview "github.com/Clyra-AI/axym/core/review"
	reviewexport "github.com/Clyra-AI/axym/core/review/export"
	"github.com/spf13/cobra"
)

func newReviewCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	var dateValue string
	var storeDir string
	var format string

	cmd := &cobra.Command{
		Use:   "review",
		Short: "Generate a deterministic daily review pack",
		RunE: func(cmd *cobra.Command, args []string) error {
			dateValue = strings.TrimSpace(dateValue)
			if dateValue == "" {
				return emitReviewInvalidInput("date is required (YYYY-MM-DD)", stdout, stderr, global)
			}
			reviewDate, err := time.Parse("2006-01-02", dateValue)
			if err != nil {
				return emitReviewInvalidInput("date must use YYYY-MM-DD", stdout, stderr, global)
			}
			normalizedFormat := strings.ToLower(strings.TrimSpace(format))
			if normalizedFormat == "" {
				normalizedFormat = "json"
			}
			switch normalizedFormat {
			case "json", "csv", "pdf":
			default:
				return emitReviewInvalidInput("format must be one of: json,csv,pdf", stdout, stderr, global)
			}

			pack, err := corereview.Build(corereview.Request{StoreDir: storeDir, Date: reviewDate})
			if err != nil {
				return emitReviewError(err, stdout, stderr, global)
			}
			if global.JSON {
				return printJSON(stdout, envelope{OK: true, Command: "review", Data: pack})
			}
			if global.Quiet {
				return nil
			}

			var payload []byte
			switch normalizedFormat {
			case "csv":
				payload, err = reviewexport.CSV(pack)
			case "pdf":
				payload = reviewexport.PDF(pack)
			default:
				payload, err = reviewexport.JSON(pack)
			}
			if err != nil {
				return emitReviewError(err, stdout, stderr, global)
			}
			_, _ = stdout.Write(payload)
			return nil
		},
	}

	cmd.Flags().StringVar(&dateValue, "date", "", "Review date (UTC) in YYYY-MM-DD")
	cmd.Flags().StringVar(&storeDir, "store-dir", ".axym", "Path to local chain store")
	cmd.Flags().StringVar(&format, "format", "json", "Output format for non-json mode: json|csv|pdf")
	return cmd
}

func emitReviewError(err error, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	var reviewErr *corereview.Error
	if errors.As(err, &reviewErr) {
		if global.JSON {
			_ = printJSON(stdout, envelope{
				OK:      false,
				Command: "review",
				Error: &errorEnvelope{
					Reason:  reviewErr.ReasonCode,
					Message: reviewErr.Message,
				},
			})
		} else if !global.Quiet {
			_, _ = fmt.Fprintln(stderr, reviewErr.Error())
		}
		return &cliError{code: exitRuntimeFailure, msg: reviewErr.Error()}
	}

	if global.JSON {
		_ = printJSON(stdout, envelope{
			OK:      false,
			Command: "review",
			Error:   &errorEnvelope{Reason: "runtime_failure", Message: err.Error()},
		})
	}
	return &cliError{code: exitRuntimeFailure, msg: err.Error()}
}

func emitReviewInvalidInput(message string, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	if global.JSON {
		_ = printJSON(stdout, envelope{
			OK:      false,
			Command: "review",
			Error:   &errorEnvelope{Reason: "invalid_input", Message: message},
		})
	} else if !global.Quiet {
		_, _ = fmt.Fprintln(stderr, message)
	}
	return &cliError{code: exitInvalidInput, msg: message}
}
