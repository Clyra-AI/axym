package wrkr

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ingestwrkr "github.com/Clyra-AI/axym/core/ingest/wrkr"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/proof"
)

func TestIdempotentReingestDoesNotDuplicateChainSideEffects(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	st, err := store.New(store.Config{RootDir: storeDir})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	input := filepath.Join(root, "wrkr-proof-records.jsonl")
	records := []*proof.Record{
		newRecord(t, "finding-a", "agent-a", "read", time.Date(2026, 2, 28, 19, 0, 0, 0, time.UTC)),
		newRecord(t, "finding-b", "agent-a", "write", time.Date(2026, 2, 28, 19, 1, 0, 0, time.UTC)),
	}
	writeJSONL(t, input, records)

	first, err := ingestwrkr.Ingest(context.Background(), ingestwrkr.Request{
		Store:      st,
		StateDir:   root,
		InputPaths: []string{input},
	})
	if err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	second, err := ingestwrkr.Ingest(context.Background(), ingestwrkr.Request{
		Store:      st,
		StateDir:   root,
		InputPaths: []string{input},
	})
	if err != nil {
		t.Fatalf("second ingest: %v", err)
	}

	if first.Appended != 2 {
		t.Fatalf("first append mismatch: %+v", first)
	}
	if second.Appended != 0 || second.Deduped != 2 {
		t.Fatalf("second ingest should dedupe all records: %+v", second)
	}

	chain, err := st.LoadChain()
	if err != nil {
		t.Fatalf("LoadChain: %v", err)
	}
	if len(chain.Records) != 2 {
		t.Fatalf("chain record count mismatch: got %d", len(chain.Records))
	}
}

func newRecord(t *testing.T, findingID string, principal string, privilege string, ts time.Time) *proof.Record {
	t.Helper()
	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     ts,
		Source:        "wrkr",
		SourceProduct: "wrkr",
		AgentID:       principal,
		Type:          "scan_finding",
		Event: map[string]any{
			"finding_id":   findingID,
			"principal_id": principal,
			"privilege":    privilege,
			"approved":     true,
		},
		Metadata: map[string]any{
			"principal_id": principal,
			"scope":        privilege,
		},
		Controls: proof.Controls{
			PermissionsEnforced: true,
		},
	})
	if err != nil {
		t.Fatalf("proof.NewRecord: %v", err)
	}
	return record
}

func writeJSONL(t *testing.T, path string, records []*proof.Record) {
	t.Helper()
	lines := make([]string, 0, len(records))
	for _, record := range records {
		raw, err := json.Marshal(record)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		lines = append(lines, string(raw))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}
