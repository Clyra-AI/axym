package ingest

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ingestgait "github.com/Clyra-AI/axym/core/ingest/gait"
	ingestwrkr "github.com/Clyra-AI/axym/core/ingest/wrkr"
	"github.com/Clyra-AI/axym/core/normalize"
	"github.com/Clyra-AI/axym/core/store"
	coreverify "github.com/Clyra-AI/axym/core/verify"
	"github.com/Clyra-AI/proof"
)

func TestMixedSourceChainVerifiesWithAxymAndProof(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	st, err := store.New(store.Config{RootDir: storeDir})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	wrkrInput := filepath.Join(root, "wrkr.jsonl")
	writeWrkrRecord(t, wrkrInput)
	if _, err := ingestwrkr.Ingest(context.Background(), ingestwrkr.Request{
		Store:      st,
		StateDir:   root,
		InputPaths: []string{wrkrInput},
	}); err != nil {
		t.Fatalf("wrkr ingest: %v", err)
	}

	gaitPackDir := filepath.Join(root, "gait-pack")
	if err := os.MkdirAll(gaitPackDir, 0o700); err != nil {
		t.Fatalf("mkdir gait pack: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gaitPackDir, "native_records.jsonl"), []byte(`{"type":"trace","timestamp":"2026-02-28T23:31:00Z","agent_id":"agent://executor","event":{"tool_name":"planner","actor_identity":"agent://requester"},"metadata":{"session_id":"mixed-session"}}`+"\n"), 0o600); err != nil {
		t.Fatalf("write gait native file: %v", err)
	}
	if _, err := ingestgait.Ingest(context.Background(), ingestgait.Request{
		Store:      st,
		InputPaths: []string{gaitPackDir},
	}); err != nil {
		t.Fatalf("gait ingest: %v", err)
	}

	axymVerify, err := coreverify.VerifyChainFromStoreDir(storeDir)
	if err != nil {
		t.Fatalf("axym verify chain: %v", err)
	}
	if !axymVerify.Intact || axymVerify.Count != 2 {
		t.Fatalf("axym verify mismatch: %+v", axymVerify)
	}

	chain, err := st.LoadChain()
	if err != nil {
		t.Fatalf("load chain: %v", err)
	}
	proofVerify, err := proof.VerifyChain(chain)
	if err != nil {
		t.Fatalf("proof verify chain: %v", err)
	}
	if !proofVerify.Intact || proofVerify.Count != 2 {
		t.Fatalf("proof verify mismatch: %+v", proofVerify)
	}

	firstView := normalize.IdentityViewFromRecord(&chain.Records[0])
	secondView := normalize.IdentityViewFromRecord(&chain.Records[1])
	if firstView.ActorIdentity == "" || secondView.ActorIdentity == "" {
		t.Fatalf("expected normalized identities across mixed source chain: first=%+v second=%+v", firstView, secondView)
	}
}

func writeWrkrRecord(t *testing.T, path string) {
	t.Helper()
	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     time.Date(2026, 2, 28, 23, 30, 0, 0, time.UTC),
		Source:        "wrkr",
		SourceProduct: "wrkr",
		Type:          "scan_finding",
		AgentID:       "agent-a",
		Event: map[string]any{
			"finding_id":   "mixed-finding-1",
			"principal_id": "agent-a",
			"privilege":    "read",
			"approved":     true,
		},
		Metadata: map[string]any{
			"principal_id": "agent-a",
			"scope":        "read",
			"session_id":   "mixed-session",
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
