package replay

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/store"
)

func TestRunUsesPerRunDedupeIdentity(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	times := []time.Time{
		time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 15, 16, 0, 0, 500_000_000, time.UTC),
	}
	index := 0
	clock := func() time.Time {
		if index >= len(times) {
			return times[len(times)-1]
		}
		value := times[index]
		index++
		return value
	}

	first, err := Run(Request{Model: "payments-agent", Tier: "A", StoreDir: storeDir, Now: clock})
	if err != nil {
		t.Fatalf("first replay run: %v", err)
	}
	second, err := Run(Request{Model: "payments-agent", Tier: "A", StoreDir: storeDir, Now: clock})
	if err != nil {
		t.Fatalf("second replay run: %v", err)
	}
	if first.RecordCount != 1 {
		t.Fatalf("first record count mismatch: %+v", first)
	}
	if second.RecordCount != 2 {
		t.Fatalf("second run should append instead of dedupe: %+v", second)
	}

	evidenceStore, err := store.New(store.Config{RootDir: storeDir, ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	chain, err := evidenceStore.LoadChain()
	if err != nil {
		t.Fatalf("LoadChain: %v", err)
	}
	if len(chain.Records) != 2 {
		t.Fatalf("chain length mismatch, want 2 got %d", len(chain.Records))
	}
}
