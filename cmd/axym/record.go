package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/proof"
	"github.com/spf13/cobra"
)

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
				return emitRecordInvalidInput(err.Error(), stdout, stderr, global)
			}
			st, err := store.New(store.Config{RootDir: storeDir})
			if err != nil {
				return emitRecordError(fmt.Errorf("initialize local store: %w", err), stdout, stderr, global)
			}
			result, err := st.Append(record, "record-add:"+strings.TrimSpace(record.RecordID))
			if err != nil {
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
		return nil, fmt.Errorf("input is required")
	}
	// #nosec G304 -- record input path is explicit user input.
	raw, err := os.ReadFile(trimmed)
	if err != nil {
		return nil, fmt.Errorf("read input file: %w", err)
	}
	var record proof.Record
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, fmt.Errorf("decode input json: %w", err)
	}
	if strings.TrimSpace(record.RecordID) == "" {
		return nil, fmt.Errorf("record_id is required")
	}
	if strings.TrimSpace(record.RecordVersion) == "" {
		record.RecordVersion = "v1"
	}
	if strings.TrimSpace(record.Source) == "" {
		return nil, fmt.Errorf("source is required")
	}
	if strings.TrimSpace(record.SourceProduct) == "" {
		return nil, fmt.Errorf("source_product is required")
	}
	if strings.TrimSpace(record.RecordType) == "" {
		return nil, fmt.Errorf("record_type is required")
	}
	if strings.TrimSpace(record.AgentID) == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	if record.Timestamp.IsZero() || record.Timestamp.Equal(time.Time{}) {
		return nil, fmt.Errorf("timestamp is required")
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
