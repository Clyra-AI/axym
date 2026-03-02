package ticket

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

func TestChaosRateLimitStormBoundedRetries(t *testing.T) {
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
		Adapter:     jira.NewScripted([]int{429, 429, 429, 429}),
		Store:       evidenceStore,
		DLQ:         queue,
		MaxAttempts: 3,
		Clock:       func() time.Time { return time.Date(2026, 9, 15, 14, 0, 0, 0, time.UTC) },
		Sleep:       func(context.Context, time.Duration) error { return nil },
	}
	result, err := processor.Process(context.Background(), coreticket.Request{ChangeID: "CHG-999", PayloadHash: "storm"})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if result.Attempts != 3 {
		t.Fatalf("retry attempts must be bounded: %+v", result)
	}
	if result.Status != coreticket.StatusDLQ {
		t.Fatalf("expected DLQ after sustained 429 storm: %+v", result)
	}
}
