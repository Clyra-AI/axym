package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Clyra-AI/proof"
)

func TestShouldIgnoreWindowsDirSyncError(t *testing.T) {
	t.Parallel()

	tempChild := filepath.Join(os.TempDir(), "axym-test")
	outside := filepath.Join(string(os.PathSeparator), "var", "lib", "axym")
	permErr := &os.PathError{Op: "sync", Path: tempChild, Err: os.ErrPermission}

	if !shouldIgnoreWindowsDirSyncError("windows", tempChild, permErr) {
		t.Fatal("expected windows temp permission error to be ignored")
	}
	if shouldIgnoreWindowsDirSyncError("windows", outside, permErr) {
		t.Fatal("expected non-temp path permission error to be enforced")
	}
	nonPermErr := &os.PathError{Op: "sync", Path: tempChild, Err: errors.New("other")}
	if shouldIgnoreWindowsDirSyncError("windows", tempChild, nonPermErr) {
		t.Fatal("expected non-permission error to be enforced")
	}
	if shouldIgnoreWindowsDirSyncError(runtime.GOOS, tempChild, permErr) && runtime.GOOS != "windows" {
		t.Fatal("expected non-windows runtime to enforce permission error")
	}
}

func TestAppendRoundTripsEmptyMetadata(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	s, err := New(Config{RootDir: storeDir, ComplianceMode: true})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	record := &proof.Record{
		RecordID:      "manual-001",
		RecordVersion: "v1",
		Timestamp:     time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC),
		Source:        "manual",
		SourceProduct: "axym",
		AgentID:       "agent-1",
		RecordType:    "approval",
		Event:         map[string]any{"decision": "allow"},
		Metadata:      map[string]any{},
		Controls: proof.Controls{
			PermissionsEnforced: true,
		},
	}

	if _, err := s.Append(record, "record-add:"+record.RecordID); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	chain, err := s.LoadChain()
	if err != nil {
		t.Fatalf("LoadChain() error = %v", err)
	}
	verification, err := proof.VerifyChain(chain)
	if err != nil {
		t.Fatalf("VerifyChain() error = %v", err)
	}
	if !verification.Intact {
		t.Fatalf("expected intact chain, got %+v", verification)
	}

	raw, err := os.ReadFile(filepath.Join(storeDir, defaultChainFile))
	if err != nil {
		t.Fatalf("ReadFile(chain.json) error = %v", err)
	}
	var persisted map[string]any
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatalf("decode persisted chain: %v", err)
	}
	records, ok := persisted["records"].([]any)
	if !ok || len(records) != 1 {
		t.Fatalf("unexpected persisted records payload: %#v", persisted["records"])
	}
	first, ok := records[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected first record payload: %#v", records[0])
	}
	if _, exists := first["metadata"]; exists {
		t.Fatalf("expected canonical persisted record to omit empty metadata, got %#v", first["metadata"])
	}
}

func TestOpenReadOnlyDoesNotCreateStoreArtifacts(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	s, err := OpenReadOnly(Config{RootDir: storeDir})
	if err != nil {
		t.Fatalf("OpenReadOnly() error = %v", err)
	}
	if _, err := os.Stat(storeDir); !os.IsNotExist(err) {
		t.Fatalf("expected read-only open to avoid creating store dir, got err=%v", err)
	}

	chain, err := s.LoadChain()
	if err != nil {
		t.Fatalf("LoadChain() error = %v", err)
	}
	if len(chain.Records) != 0 {
		t.Fatalf("expected empty chain from missing store, got %d records", len(chain.Records))
	}
}

func TestReadOnlyStoreRejectsAppend(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	s, err := OpenReadOnly(Config{RootDir: storeDir})
	if err != nil {
		t.Fatalf("OpenReadOnly() error = %v", err)
	}

	record := &proof.Record{
		RecordID:      "manual-002",
		RecordVersion: "v1",
		Timestamp:     time.Date(2026, 3, 18, 1, 0, 0, 0, time.UTC),
		Source:        "manual",
		SourceProduct: "axym",
		RecordType:    "approval",
		Event:         map[string]any{"decision": "allow"},
		Controls:      proof.Controls{PermissionsEnforced: true},
	}
	if _, err := s.Append(record, "record-add:"+record.RecordID); err == nil {
		t.Fatal("expected read-only store append to fail")
	}
}
