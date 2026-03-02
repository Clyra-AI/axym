package stitch

import (
	"testing"
	"time"

	"github.com/Clyra-AI/proof"
)

func TestAnalyzeDetectsSessionGapWithExactWindow(t *testing.T) {
	t.Parallel()

	records := []proof.Record{
		buildRecord(t, "session-a", time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)),
		buildRecord(t, "session-a", time.Date(2026, 2, 28, 10, 5, 0, 0, time.UTC)),
		buildRecord(t, "session-b", time.Date(2026, 2, 28, 11, 30, 0, 0, time.UTC)),
	}
	result := Analyze(records, Config{MaxGap: 30 * time.Minute})
	if result.Intact {
		t.Fatalf("expected session gap result: %+v", result)
	}
	if len(result.Gaps) != 1 {
		t.Fatalf("expected exactly one gap: %+v", result.Gaps)
	}
	gap := result.Gaps[0]
	if gap.ReasonCode != ReasonChainSessionGap {
		t.Fatalf("reason mismatch: %+v", gap)
	}
	if gap.GapStart != "2026-02-28T10:05:00Z" || gap.GapEnd != "2026-02-28T11:30:00Z" {
		t.Fatalf("window mismatch: %+v", gap)
	}
}

func TestAnalyzeIntactForContinuousTimeline(t *testing.T) {
	t.Parallel()

	records := []proof.Record{
		buildRecord(t, "session-a", time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)),
		buildRecord(t, "session-a", time.Date(2026, 2, 28, 10, 10, 0, 0, time.UTC)),
		buildRecord(t, "session-a", time.Date(2026, 2, 28, 10, 20, 0, 0, time.UTC)),
	}
	result := Analyze(records, Config{MaxGap: 30 * time.Minute})
	if !result.Intact || len(result.Gaps) != 0 {
		t.Fatalf("expected intact chain: %+v", result)
	}
}

func buildRecord(t *testing.T, sessionID string, ts time.Time) proof.Record {
	t.Helper()
	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     ts,
		Source:        "gait",
		SourceProduct: "gait",
		Type:          "tool_invocation",
		Event: map[string]any{
			"tool_name": "planner",
		},
		Metadata: map[string]any{
			"session_id": sessionID,
		},
		Controls: proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("proof.NewRecord: %v", err)
	}
	return *record
}
