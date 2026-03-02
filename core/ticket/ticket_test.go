package ticket_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/store"
	coreticket "github.com/Clyra-AI/axym/core/ticket"
	"github.com/Clyra-AI/axym/core/ticket/dlq"
	"github.com/Clyra-AI/axym/core/ticket/jira"
	"github.com/Clyra-AI/axym/core/ticket/servicenow"
)

func TestProcessorRateLimitRecovery(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	evidenceStore, err := store.New(store.Config{RootDir: storeDir})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	queue, err := dlq.New(filepath.Join(storeDir, "dlq"))
	if err != nil {
		t.Fatalf("dlq.New: %v", err)
	}
	processor := coreticket.Processor{
		Adapter:     jira.NewScripted([]int{429, 429, 200}),
		Store:       evidenceStore,
		DLQ:         queue,
		MaxAttempts: 3,
		Clock:       func() time.Time { return time.Date(2026, 9, 15, 10, 0, 0, 0, time.UTC) },
		Sleep:       func(context.Context, time.Duration) error { return nil },
	}

	result, err := processor.Process(context.Background(), coreticket.Request{
		ChangeID:    "CHG-100",
		PayloadHash: "abc",
		OpenedAt:    time.Date(2026, 9, 15, 9, 58, 0, 0, time.UTC),
		SLA:         5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if result.Status != coreticket.StatusAttached {
		t.Fatalf("status mismatch: %+v", result)
	}
	if result.Attempts != 3 {
		t.Fatalf("attempt mismatch: %+v", result)
	}
	if len(result.ReasonCodes) == 0 || result.ReasonCodes[0] != coreticket.ReasonRateLimited {
		t.Fatalf("expected stable reason codes to include rate limit: %+v", result)
	}
	if !result.SLAWithin {
		t.Fatalf("expected SLA within: %+v", result)
	}

	chain, err := evidenceStore.LoadChain()
	if err != nil {
		t.Fatalf("LoadChain: %v", err)
	}
	if len(chain.Records) != 1 {
		t.Fatalf("expected emitted ticket record, got=%d", len(chain.Records))
	}
	if got, _ := chain.Records[0].Event["kind"].(string); got != "ticket_attachment" {
		t.Fatalf("record kind mismatch: %+v", chain.Records[0].Event)
	}
}

func TestProcessorSustainedFailureToDLQ(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	evidenceStore, err := store.New(store.Config{RootDir: storeDir})
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
		Clock:       func() time.Time { return time.Date(2026, 9, 15, 11, 0, 0, 0, time.UTC) },
		Sleep:       func(context.Context, time.Duration) error { return nil },
	}

	result, err := processor.Process(context.Background(), coreticket.Request{
		ChangeID:    "CHG-200",
		PayloadHash: "xyz",
		OpenedAt:    time.Date(2026, 9, 15, 10, 45, 0, 0, time.UTC),
		SLA:         10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if result.Status != coreticket.StatusDLQ {
		t.Fatalf("status mismatch: %+v", result)
	}
	if result.DLQPath == "" {
		t.Fatalf("expected dlq path in result: %+v", result)
	}
	entries, err := dlq.ReadAll(result.DLQPath)
	if err != nil {
		t.Fatalf("ReadAll dlq: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one dlq entry, got=%d", len(entries))
	}
	if !contains(result.ReasonCodes, coreticket.ReasonDLQ) || !contains(result.ReasonCodes, coreticket.ReasonRemoteError) {
		t.Fatalf("unexpected reason codes: %+v", result.ReasonCodes)
	}
	if result.SLAWithin {
		t.Fatalf("expected breached SLA for delayed failure: %+v", result)
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

func TestProcessorUsesBackoffForRetries(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	evidenceStore, err := store.New(store.Config{RootDir: storeDir})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	queue, err := dlq.New(filepath.Join(storeDir, "dlq"))
	if err != nil {
		t.Fatalf("dlq.New: %v", err)
	}
	delays := make([]time.Duration, 0)
	processor := coreticket.Processor{
		Adapter:     jira.NewScripted([]int{429, 429, 200}),
		Store:       evidenceStore,
		DLQ:         queue,
		MaxAttempts: 3,
		Clock:       func() time.Time { return time.Date(2026, 9, 15, 12, 0, 0, 0, time.UTC) },
		RetryBackoff: func(attempt int, reasonCode string) time.Duration {
			return time.Duration(attempt) * 10 * time.Millisecond
		},
		Sleep: func(_ context.Context, delay time.Duration) error {
			delays = append(delays, delay)
			return nil
		},
	}

	result, err := processor.Process(context.Background(), coreticket.Request{
		ChangeID:    "CHG-300",
		PayloadHash: "retry",
	})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if result.Status != coreticket.StatusAttached {
		t.Fatalf("status mismatch: %+v", result)
	}
	if len(delays) != 2 || delays[0] != 10*time.Millisecond || delays[1] != 20*time.Millisecond {
		t.Fatalf("unexpected retry delays: %+v", delays)
	}
}

func TestProcessorEvaluatesSLAAtCompletionTime(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	evidenceStore, err := store.New(store.Config{RootDir: storeDir})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	queue, err := dlq.New(filepath.Join(storeDir, "dlq"))
	if err != nil {
		t.Fatalf("dlq.New: %v", err)
	}
	moments := []time.Time{
		time.Date(2026, 9, 15, 13, 0, 0, 0, time.UTC),  // start
		time.Date(2026, 9, 15, 13, 11, 0, 0, time.UTC), // completion
	}
	clockIndex := 0
	clock := func() time.Time {
		if clockIndex >= len(moments) {
			return moments[len(moments)-1]
		}
		current := moments[clockIndex]
		clockIndex++
		return current
	}
	processor := coreticket.Processor{
		Adapter:     servicenow.NewScripted([]int{500, 500, 500}),
		Store:       evidenceStore,
		DLQ:         queue,
		MaxAttempts: 3,
		Clock:       clock,
		Sleep:       func(context.Context, time.Duration) error { return nil },
	}

	result, err := processor.Process(context.Background(), coreticket.Request{
		ChangeID:    "CHG-400",
		PayloadHash: "sla",
		OpenedAt:    time.Date(2026, 9, 15, 13, 0, 0, 0, time.UTC),
		SLA:         10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Process: %v", err)
	}
	if result.SLAWithin {
		t.Fatalf("expected SLA breach at completion time: %+v", result)
	}
}
