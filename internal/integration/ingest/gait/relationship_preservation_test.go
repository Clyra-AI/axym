package gait

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	ingestgait "github.com/Clyra-AI/axym/core/ingest/gait"
	"github.com/Clyra-AI/axym/core/store"
)

func TestRelationshipEnvelopePreservedDuringGaitTranslation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	packDir := filepath.Join(root, "pack")
	storeDir := filepath.Join(root, "store")
	if err := os.MkdirAll(packDir, 0o700); err != nil {
		t.Fatalf("MkdirAll pack: %v", err)
	}
	native := map[string]any{
		"type":      "trace",
		"timestamp": "2026-02-28T22:00:00Z",
		"agent_id":  "agent-a",
		"event": map[string]any{
			"tool_name": "planner",
		},
		"relationship": map[string]any{
			"parent_ref": map[string]any{
				"kind": "trace",
				"id":   "parent-1",
			},
			"entity_refs": []map[string]any{
				{
					"kind": "resource",
					"id":   "resource-1",
				},
			},
		},
	}
	raw, err := json.Marshal(native)
	if err != nil {
		t.Fatalf("marshal native: %v", err)
	}
	if err := os.WriteFile(filepath.Join(packDir, "native_records.jsonl"), append(raw, '\n'), 0o600); err != nil {
		t.Fatalf("write native file: %v", err)
	}

	st, err := store.New(store.Config{RootDir: storeDir})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	_, err = ingestgait.Ingest(context.Background(), ingestgait.Request{
		Store:      st,
		InputPaths: []string{packDir},
	})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}

	chain, err := st.LoadChain()
	if err != nil {
		t.Fatalf("LoadChain: %v", err)
	}
	if len(chain.Records) != 1 {
		t.Fatalf("record count mismatch: got %d", len(chain.Records))
	}
	relationship := chain.Records[0].Relationship
	if relationship == nil {
		t.Fatalf("expected relationship to be preserved")
	}
	if relationship.ParentRef == nil || relationship.ParentRef.ID != "parent-1" {
		t.Fatalf("parent ref mismatch: %+v", relationship)
	}
	foundResource := false
	for _, ref := range relationship.EntityRefs {
		if ref.ID == "resource-1" {
			foundResource = true
			break
		}
	}
	if !foundResource {
		t.Fatalf("expected resource relationship ref to be preserved: %+v", relationship.EntityRefs)
	}
}
