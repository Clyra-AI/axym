package gait

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/proof"
)

func TestIngestNoInputReturnsNoInputReason(t *testing.T) {
	t.Parallel()

	st, err := store.New(store.Config{RootDir: filepath.Join(t.TempDir(), "store")})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	result, err := Ingest(context.Background(), Request{Store: st})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if len(result.ReasonCodes) != 1 || result.ReasonCodes[0] != ReasonNoInput {
		t.Fatalf("reason mismatch: %+v", result)
	}
}

func TestIngestMixesPassthroughAndNativeRecords(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	st, err := store.New(store.Config{RootDir: filepath.Join(root, "store")})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	packDir := filepath.Join(root, "pack")
	if err := os.MkdirAll(packDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeProofJSONL(t, filepath.Join(packDir, "proof_records.jsonl"))
	if err := os.WriteFile(filepath.Join(packDir, "native_records.jsonl"), []byte(`{"type":"trace","timestamp":"2026-02-28T21:30:00Z","event":{"tool_name":"planner"}}`+"\n"), 0o600); err != nil {
		t.Fatalf("write native: %v", err)
	}

	result, err := Ingest(context.Background(), Request{
		Store:      st,
		InputPaths: []string{packDir},
	})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if result.Appended != 2 || result.Passthrough != 1 || result.Translated != 1 {
		t.Fatalf("result mismatch: %+v", result)
	}
}

func writeProofJSONL(t *testing.T, path string) {
	t.Helper()
	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     time.Date(2026, 2, 28, 21, 29, 0, 0, time.UTC),
		Source:        "wrkr",
		SourceProduct: "wrkr",
		Type:          "approval",
		Event: map[string]any{
			"decision": "allow",
		},
		Controls: proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("proof.NewRecord: %v", err)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimSpace(string(raw))+"\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}
