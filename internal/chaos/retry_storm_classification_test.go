package chaos

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/store"
	coreticket "github.com/Clyra-AI/axym/core/ticket"
	"github.com/Clyra-AI/axym/core/ticket/dlq"
	"github.com/Clyra-AI/axym/core/ticket/jira"
)

func TestChaosRetryStormClassification(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	evidenceStore, err := store.New(store.Config{RootDir: storeDir, ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	queue, err := dlq.New(filepath.Join(storeDir, "dlq"))
	if err != nil {
		t.Fatalf("dlq.New: %v", err)
	}
	processor := coreticket.Processor{
		Adapter:     jira.NewScripted([]int{429, 429, 429}),
		Store:       evidenceStore,
		DLQ:         queue,
		MaxAttempts: 3,
		Clock:       func() time.Time { return time.Date(2026, 9, 15, 14, 0, 0, 0, time.UTC) },
		Sleep:       func(context.Context, time.Duration) error { return nil },
	}
	result, err := processor.Process(context.Background(), coreticket.Request{
		ChangeID:    "CHG-999",
		PayloadHash: "storm",
		OpenedAt:    time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC),
		SLA:         30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if result.Status != coreticket.StatusDLQ {
		t.Fatalf("expected dlq status, got %+v", result)
	}
	wantReasons := []string{coreticket.ReasonDLQ, coreticket.ReasonRateLimited}
	if diffReasonCodes(result.ReasonCodes, wantReasons) {
		t.Fatalf("reason code mismatch: got=%v want=%v", result.ReasonCodes, wantReasons)
	}
	entries, err := dlq.ReadAll(queue.Path())
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("dlq entry count mismatch: %+v", entries)
	}
	if diffReasonCodes(entries[0].ReasonCodes, wantReasons) {
		t.Fatalf("dlq reason code mismatch: got=%v want=%v", entries[0].ReasonCodes, wantReasons)
	}
}

func diffReasonCodes(got []string, want []string) bool {
	if len(got) != len(want) {
		return true
	}
	for i := range got {
		if got[i] != want[i] {
			return true
		}
	}
	return false
}
