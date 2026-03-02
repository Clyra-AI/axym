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

func TestHardeningRateLimitRetry(t *testing.T) {
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
		Adapter:     jira.NewScripted([]int{429, 200}),
		Store:       evidenceStore,
		DLQ:         queue,
		MaxAttempts: 3,
		Clock:       func() time.Time { return time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC) },
		Sleep:       func(context.Context, time.Duration) error { return nil },
	}
	result, err := processor.Process(context.Background(), coreticket.Request{
		ChangeID:    "CHG-777",
		PayloadHash: "fixture",
		OpenedAt:    time.Date(2026, 9, 15, 11, 59, 0, 0, time.UTC),
		SLA:         5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if result.Status != coreticket.StatusAttached || result.Attempts != 2 {
		t.Fatalf("unexpected retry outcome: %+v", result)
	}
	if !contains(result.ReasonCodes, coreticket.ReasonRateLimited) {
		t.Fatalf("expected rate-limit reason code stability: %+v", result.ReasonCodes)
	}
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
