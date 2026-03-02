package pack

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Clyra-AI/proof"
)

func TestReadDirectoryPack(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeProofFile(t, filepath.Join(dir, "proof_records.jsonl"))
	if err := os.WriteFile(filepath.Join(dir, "native_records.jsonl"), []byte(`{"type":"trace","timestamp":"2026-02-28T21:00:00Z","event":{"tool_name":"planner"}}`+"\n"), 0o600); err != nil {
		t.Fatalf("write native file: %v", err)
	}

	result, err := Read(dir)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(result.ProofRecords) != 1 || len(result.NativeRecords) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestReadZipPack(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	zipPath := filepath.Join(root, "pack.zip")

	file, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	writer := zip.NewWriter(file)

	proofEntry, err := writer.Create("proof_records.jsonl")
	if err != nil {
		t.Fatalf("create proof entry: %v", err)
	}
	proofLine := buildProofLine(t)
	if _, err := proofEntry.Write([]byte(proofLine + "\n")); err != nil {
		t.Fatalf("write proof entry: %v", err)
	}

	nativeEntry, err := writer.Create("native_records.jsonl")
	if err != nil {
		t.Fatalf("create native entry: %v", err)
	}
	if _, err := nativeEntry.Write([]byte(`{"type":"approval_token","timestamp":"2026-02-28T21:01:00Z","event":{"decision":"allow"}}` + "\n")); err != nil {
		t.Fatalf("write native entry: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close zip file: %v", err)
	}

	result, err := Read(zipPath)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(result.ProofRecords) != 1 || len(result.NativeRecords) != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestReadDirectoryPackRejectsEmptyDirectory(t *testing.T) {
	t.Parallel()

	_, err := Read(t.TempDir())
	if err == nil {
		t.Fatal("expected error for empty gait pack directory")
	}
}

func writeProofFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(buildProofLine(t)+"\n"), 0o600); err != nil {
		t.Fatalf("write proof file: %v", err)
	}
}

func buildProofLine(t *testing.T) string {
	t.Helper()
	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     time.Date(2026, 2, 28, 21, 0, 0, 0, time.UTC),
		Source:        "gait",
		SourceProduct: "gait",
		Type:          "approval",
		Event: map[string]any{
			"decision": "allow",
		},
		Controls: proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("NewRecord: %v", err)
	}
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	return strings.TrimSpace(string(raw))
}
