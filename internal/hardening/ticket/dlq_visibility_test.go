package ticket

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/review"
	"github.com/Clyra-AI/axym/core/store"
	coreticket "github.com/Clyra-AI/axym/core/ticket"
	"github.com/Clyra-AI/axym/core/ticket/dlq"
	"github.com/Clyra-AI/axym/core/ticket/servicenow"
)

func TestHardeningDLQVisibilityInReview(t *testing.T) {
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
		Adapter:     servicenow.NewScripted([]int{500, 500, 500}),
		Store:       evidenceStore,
		DLQ:         queue,
		MaxAttempts: 3,
		Clock:       func() time.Time { return time.Date(2026, 9, 15, 13, 0, 0, 0, time.UTC) },
	}
	result, err := processor.Process(context.Background(), coreticket.Request{
		ChangeID:    "CHG-888",
		PayloadHash: "fixture-2",
		OpenedAt:    time.Date(2026, 9, 15, 12, 30, 0, 0, time.UTC),
		SLA:         10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if result.Status != coreticket.StatusDLQ {
		t.Fatalf("expected DLQ status: %+v", result)
	}

	pack, err := review.Build(review.Request{
		StoreDir: storeDir,
		Date:     time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("review.Build: %v", err)
	}
	if countByName(pack.AttachStatus, "dlq") != 1 {
		t.Fatalf("expected dlq attach status in review pack: %+v", pack.AttachStatus)
	}
	if countByName(pack.Exceptions, review.ExceptionAttach) != 1 {
		t.Fatalf("expected attach exception in review pack: %+v", pack.Exceptions)
	}
}

func countByName(rows []review.Count, name string) int {
	for _, row := range rows {
		if row.Name == name {
			return row.Count
		}
	}
	return 0
}
