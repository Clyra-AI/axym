package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Clyra-AI/axym/core/store"
	recordschema "github.com/Clyra-AI/axym/schemas/v1/record"
	"github.com/Clyra-AI/proof"
	"github.com/spf13/cobra"
)

type recordInputError struct {
	kind string
	err  error
}

func (e *recordInputError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e *recordInputError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

func newRecordCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "record",
		Short: "Manage proof records",
	}
	cmd.AddCommand(newRecordAddCmd(stdout, stderr, global))
	return cmd
}

func newRecordAddCmd(stdout io.Writer, stderr io.Writer, global *globalFlags) *cobra.Command {
	var inputPath string
	var storeDir string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Append a record to the local proof chain",
		RunE: func(cmd *cobra.Command, args []string) error {
			record, err := decodeRecord(inputPath)
			if err != nil {
				var inputErr *recordInputError
				if errors.As(err, &inputErr) && inputErr.kind == "schema_violation" {
					return emitRecordSchemaViolation(err.Error(), stdout, stderr, global)
				}
				return emitRecordInvalidInput(err.Error(), stdout, stderr, global)
			}
			st, err := store.New(store.Config{RootDir: storeDir})
			if err != nil {
				return emitRecordError(fmt.Errorf("initialize local store: %w", err), stdout, stderr, global)
			}
			result, err := st.Append(record, "record-add:"+strings.TrimSpace(record.RecordID))
			if err != nil {
				var validationErr *store.ValidationError
				if errors.As(err, &validationErr) {
					return emitRecordSchemaViolation(validationErr.Error(), stdout, stderr, global)
				}
				return emitRecordError(fmt.Errorf("append record: %w", err), stdout, stderr, global)
			}
			data := map[string]any{
				"record_id":    result.RecordID,
				"appended":     result.Appended,
				"deduped":      result.Deduped,
				"head_hash":    result.HeadHash,
				"record_count": result.RecordCount,
			}
			if global.JSON {
				return printJSON(stdout, envelope{OK: true, Command: "record", Data: data})
			}
			printText(stdout, fmt.Sprintf("record add: appended=%t deduped=%t record_id=%s", result.Appended, result.Deduped, result.RecordID), global.Quiet)
			return nil
		},
	}
	cmd.Flags().StringVar(&inputPath, "input", "", "Path to a JSON proof record payload")
	cmd.Flags().StringVar(&storeDir, "store-dir", ".axym", "Path to local chain store")
	return cmd
}

func decodeRecord(path string) (*proof.Record, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, &recordInputError{kind: "invalid_input", err: fmt.Errorf("input is required")}
	}
	// #nosec G304 -- record input path is explicit user input.
	raw, err := os.ReadFile(trimmed)
	if err != nil {
		return nil, &recordInputError{kind: "invalid_input", err: fmt.Errorf("read input file: %w", err)}
	}
	normalizedRaw, err := recordschema.NormalizeManualInput(raw)
	if err != nil {
		return nil, &recordInputError{kind: "invalid_input", err: err}
	}
	if err := recordschema.ValidateManualInput(normalizedRaw); err != nil {
		return nil, &recordInputError{kind: "schema_violation", err: fmt.Errorf("manual record contract validation failed: %w", err)}
	}
	var record proof.Record
	if err := json.Unmarshal(normalizedRaw, &record); err != nil {
		return nil, &recordInputError{kind: "schema_violation", err: fmt.Errorf("decode validated input json: %w", err)}
	}
	if record.Event == nil {
		record.Event = map[string]any{}
	}
	if record.Metadata == nil {
		record.Metadata = map[string]any{}
	}
	return &record, nil
}

func emitRecordError(err error, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	if global.JSON {
		_ = printJSON(stdout, envelope{
			OK:      false,
			Command: "record",
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

func emitRecordInvalidInput(message string, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	if global.JSON {
		_ = printJSON(stdout, envelope{
			OK:      false,
			Command: "record",
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

func emitRecordSchemaViolation(message string, stdout io.Writer, stderr io.Writer, global *globalFlags) error {
	if global.JSON {
		_ = printJSON(stdout, envelope{
			OK:      false,
			Command: "record",
			Error: &errorEnvelope{
				Reason:  "schema_violation",
				Message: message,
			},
		})
	} else if !global.Quiet {
		_, _ = fmt.Fprintln(stderr, message)
	}
	return &cliError{code: exitPolicyViolation, msg: message}
}
