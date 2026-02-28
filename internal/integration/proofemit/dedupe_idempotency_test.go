package proofemit

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/normalize"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/proofemit"
	"github.com/Clyra-AI/axym/core/store"
)

func TestDedupeIdempotency(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "state")
	s, err := store.New(store.Config{RootDir: storeDir, ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	emitter := proofemit.Emitter{Store: s, SinkMode: sink.ModeFailClosed}

	input := proofemit.EmitInput{Normalized: normalize.Input{
		SourceType:    "mcp",
		Source:        "mcp-runtime",
		SourceProduct: "axym",
		RecordType:    "tool_invocation",
		Timestamp:     time.Date(2026, 2, 28, 14, 0, 0, 0, time.UTC),
		Event: map[string]any{
			"tool_name": "fetch_url",
			"arg":       "https://example.com",
		},
		Controls: normalize.Controls{PermissionsEnforced: true},
	}}

	first, err := emitter.Emit(input)
	if err != nil {
		t.Fatalf("first emit error = %v", err)
	}
	second, err := emitter.Emit(input)
	if err != nil {
		t.Fatalf("second emit error = %v", err)
	}

	if !first.Appended {
		t.Fatal("first emit should append")
	}
	if !second.Deduped {
		t.Fatal("second emit should dedupe")
	}

	chain, err := s.LoadChain()
	if err != nil {
		t.Fatalf("LoadChain() error = %v", err)
	}
	if len(chain.Records) != 1 {
		t.Fatalf("chain length mismatch: got %d want 1", len(chain.Records))
	}
	if chain.Records[0].Integrity.PreviousRecordHash != "" {
		t.Fatalf("first record previous hash should be empty, got %q", chain.Records[0].Integrity.PreviousRecordHash)
	}
}
