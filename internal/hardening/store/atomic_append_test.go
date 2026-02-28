package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/normalize"
	coreRecord "github.com/Clyra-AI/axym/core/record"
	"github.com/Clyra-AI/axym/core/redact"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/axym/core/store/dedupe"
)

func TestHardeningAtomicAppendNoPartialWriteOnFailure(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "store")
	if _, err := store.New(store.Config{RootDir: root, ComplianceMode: true}); err != nil {
		t.Fatalf("bootstrap store.New() error = %v", err)
	}

	failingWriter := func(path string, data []byte, fsync bool) error {
		if strings.HasSuffix(path, "chain.json") {
			return errors.New("disk full")
		}
		return store.WriteJSONAtomic(path, data, fsync)
	}
	s, err := store.New(store.Config{RootDir: root, ComplianceMode: true}, store.WithAtomicWriter(failingWriter))
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}

	r, err := coreRecord.NormalizeAndBuild(normalize.Input{
		SourceType:    "mcp",
		Source:        "mcp-runtime",
		SourceProduct: "axym",
		RecordType:    "tool_invocation",
		Timestamp:     time.Date(2026, 2, 28, 14, 10, 0, 0, time.UTC),
		Event:         map[string]any{"tool_name": "fetch"},
		Controls:      normalize.Controls{PermissionsEnforced: true},
	}, redact.Config{})
	if err != nil {
		t.Fatalf("NormalizeAndBuild() error = %v", err)
	}
	key, err := dedupe.BuildKey(r.SourceProduct, r.RecordType, r.Event)
	if err != nil {
		t.Fatalf("BuildKey() error = %v", err)
	}

	if _, err := s.Append(r, key); err == nil {
		t.Fatal("expected append failure")
	}
	if _, err := os.Stat(filepath.Join(root, "chain.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("chain file should not exist after failed append, stat err=%v", err)
	}
}
