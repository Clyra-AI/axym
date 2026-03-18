package hardening

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/normalize"
	coreRecord "github.com/Clyra-AI/axym/core/record"
	"github.com/Clyra-AI/axym/core/redact"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/axym/core/store/dedupe"
)

func TestHardeningConcurrentAppendPreservesChain(t *testing.T) {
	t.Parallel()

	evidenceStore, err := store.New(store.Config{RootDir: t.TempDir(), ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	const writers = 16
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for i := 0; i < writers; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			record, err := coreRecord.NormalizeAndBuild(normalize.Input{
				SourceType:    "mcp",
				Source:        "mcp-runtime",
				SourceProduct: "axym",
				RecordType:    "tool_invocation",
				Timestamp:     time.Date(2026, 3, 1, 10, 0, i, 0, time.UTC),
				Event:         map[string]any{"tool_name": "tool-" + strconv.Itoa(i)},
				Controls:      normalize.Controls{PermissionsEnforced: true},
			}, redact.Config{})
			if err != nil {
				errs <- err
				return
			}
			key, err := dedupe.BuildKey(record.SourceProduct, record.RecordType, record.Event)
			if err != nil {
				errs <- err
				return
			}
			if _, err := evidenceStore.Append(record, key); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("append failed: %v", err)
		}
	}

	chain, err := evidenceStore.LoadChain()
	if err != nil {
		t.Fatalf("LoadChain: %v", err)
	}
	if len(chain.Records) != writers {
		t.Fatalf("record count mismatch: got %d want %d", len(chain.Records), writers)
	}

	seenIDs := map[string]struct{}{}
	for i, record := range chain.Records {
		if record.RecordID == "" || record.Integrity.RecordHash == "" {
			t.Fatalf("record %d missing integrity fields: %+v", i, record)
		}
		if _, ok := seenIDs[record.RecordID]; ok {
			t.Fatalf("duplicate record id under contention: %s", record.RecordID)
		}
		seenIDs[record.RecordID] = struct{}{}
		if i == 0 {
			if record.Integrity.PreviousRecordHash != "" {
				t.Fatalf("first record must not claim previous hash: %+v", record.Integrity)
			}
			continue
		}
		if record.Integrity.PreviousRecordHash == "" {
			t.Fatalf("record %d missing previous hash under contention: %+v", i, record.Integrity)
		}
	}
}
