package override

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateSkipsArtifactWriteWhenDeduped(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	fixed := time.Date(2026, 9, 15, 18, 0, 0, 0, time.UTC)
	clock := func() time.Time { return fixed }

	first, err := Create(Request{
		Bundle:   "Q3-2026",
		Reason:   "fixture",
		Signer:   "ops-key",
		StoreDir: storeDir,
		Now:      clock,
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	second, err := Create(Request{
		Bundle:   "Q3-2026",
		Reason:   "fixture",
		Signer:   "ops-key",
		StoreDir: storeDir,
		Now:      clock,
	})
	if err != nil {
		t.Fatalf("second create: %v", err)
	}
	if first.RecordCount != 1 {
		t.Fatalf("first record count mismatch: %+v", first)
	}
	if second.RecordCount != 1 {
		t.Fatalf("deduped create should not append new chain record: %+v", second)
	}

	raw, err := os.ReadFile(first.ArtifactPath)
	if err != nil {
		t.Fatalf("read artifact: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 1 {
		t.Fatalf("deduped create must not append new artifact line; got=%d", len(lines))
	}
}
