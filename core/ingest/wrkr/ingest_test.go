package wrkr

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

func TestIngestNoInputReturnsDeterministicEmptyResult(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	st, err := store.New(store.Config{RootDir: root})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	result, err := Ingest(context.Background(), Request{Store: st, StateDir: root})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if result.Source != "wrkr" {
		t.Fatalf("source mismatch: %+v", result)
	}
	if len(result.ReasonCodes) != 1 || result.ReasonCodes[0] != ReasonNoInput {
		t.Fatalf("reason mismatch: %+v", result.ReasonCodes)
	}
}

func TestIngestFlagsUnapprovedPrivilegeDrift(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	st, err := store.New(store.Config{RootDir: filepath.Join(root, "store")})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	inputPath := filepath.Join(root, "wrkr.jsonl")
	records := []*proof.Record{
		newWrkrRecord(t, time.Date(2026, 2, 28, 16, 0, 0, 0, time.UTC), "agent-a", "read", true),
		newWrkrRecord(t, time.Date(2026, 2, 28, 16, 1, 0, 0, time.UTC), "agent-a", "admin", false),
	}
	writeJSONLRecords(t, inputPath, records)

	result, err := Ingest(context.Background(), Request{
		Store:      st,
		StateDir:   root,
		InputPaths: []string{inputPath},
	})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if result.Appended != 2 {
		t.Fatalf("appended mismatch: %+v", result)
	}
	if len(result.DriftGaps) != 1 {
		t.Fatalf("expected one drift gap: %+v", result.DriftGaps)
	}
	if result.DriftGaps[0].ReasonClass != ReasonPrivilegeEscalation {
		t.Fatalf("reason mismatch: %+v", result.DriftGaps[0])
	}
}

func TestIngestDedupesReingest(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	st, err := store.New(store.Config{RootDir: filepath.Join(root, "store")})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	inputPath := filepath.Join(root, "wrkr.jsonl")
	records := []*proof.Record{
		newWrkrRecord(t, time.Date(2026, 2, 28, 17, 0, 0, 0, time.UTC), "agent-a", "read", true),
		newWrkrRecord(t, time.Date(2026, 2, 28, 17, 1, 0, 0, time.UTC), "agent-a", "write", true),
	}
	writeJSONLRecords(t, inputPath, records)

	first, err := Ingest(context.Background(), Request{
		Store:      st,
		StateDir:   root,
		InputPaths: []string{inputPath},
	})
	if err != nil {
		t.Fatalf("first ingest: %v", err)
	}
	second, err := Ingest(context.Background(), Request{
		Store:      st,
		StateDir:   root,
		InputPaths: []string{inputPath},
	})
	if err != nil {
		t.Fatalf("second ingest: %v", err)
	}
	if first.Appended != 2 || second.Appended != 0 || second.Deduped != 2 {
		t.Fatalf("idempotency mismatch first=%+v second=%+v", first, second)
	}

	chain, err := st.LoadChain()
	if err != nil {
		t.Fatalf("load chain: %v", err)
	}
	if len(chain.Records) != 2 {
		t.Fatalf("chain count mismatch: got %d", len(chain.Records))
	}
}

func TestIngestRejectsUnsupportedRecordType(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	st, err := store.New(store.Config{RootDir: filepath.Join(root, "store")})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	inputPath := filepath.Join(root, "wrkr.jsonl")
	unsupported := newWrkrRecord(t, time.Date(2026, 2, 28, 18, 0, 0, 0, time.UTC), "agent-a", "read", true)
	unsupported.RecordType = "deployment"
	writeJSONLRecords(t, inputPath, []*proof.Record{unsupported})

	result, err := Ingest(context.Background(), Request{
		Store:      st,
		StateDir:   root,
		InputPaths: []string{inputPath},
	})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if result.Appended != 0 || result.Rejected != 1 {
		t.Fatalf("result mismatch: %+v", result)
	}
	if !containsReason(result.ReasonCodes, ReasonUnsupportedType) {
		t.Fatalf("missing reason code: %+v", result.ReasonCodes)
	}
}

func newWrkrRecord(t *testing.T, ts time.Time, principal string, privilege string, approved bool) *proof.Record {
	t.Helper()
	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     ts,
		Source:        "wrkr",
		SourceProduct: "wrkr",
		Type:          "scan_finding",
		AgentID:       principal,
		Event: map[string]any{
			"finding_id":   "finding-" + ts.Format("150405"),
			"principal_id": principal,
			"privilege":    privilege,
			"approved":     approved,
		},
		Metadata: map[string]any{
			"principal_id": principal,
			"scope":        privilege,
		},
		Controls: proof.Controls{
			PermissionsEnforced: true,
			ApprovedScope:       "wrkr-ingest",
		},
	})
	if err != nil {
		t.Fatalf("proof.NewRecord: %v", err)
	}
	return record
}

func writeJSONLRecords(t *testing.T, path string, records []*proof.Record) {
	t.Helper()
	lines := make([]string, 0, len(records))
	for _, record := range records {
		raw, err := json.Marshal(record)
		if err != nil {
			t.Fatalf("marshal record: %v", err)
		}
		lines = append(lines, string(raw))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
}

func containsReason(reasons []string, target string) bool {
	for _, reason := range reasons {
		if reason == target {
			return true
		}
	}
	return false
}
