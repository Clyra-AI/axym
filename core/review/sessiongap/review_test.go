package sessiongap

import (
	"testing"

	"github.com/Clyra-AI/axym/core/ingest/stitch"
)

func TestBuildSignalsGradesGapWindowsDeterministically(t *testing.T) {
	t.Parallel()

	signals := BuildSignals([]stitch.Gap{
		{
			ReasonCode:      stitch.ReasonChainSessionGap,
			GapStart:        "2026-02-28T10:00:00Z",
			GapEnd:          "2026-02-28T10:04:00Z",
			MissingSeconds:  240,
			BoundaryTrigger: "session_id_change",
		},
		{
			ReasonCode:      stitch.ReasonChainSessionGap,
			GapStart:        "2026-02-28T11:00:00Z",
			GapEnd:          "2026-02-28T11:20:00Z",
			MissingSeconds:  1200,
			BoundaryTrigger: "timestamp_discontinuity",
		},
	})
	if len(signals) != 2 {
		t.Fatalf("signal count mismatch: %+v", signals)
	}
	if signals[0].Status != "partial" || signals[0].Auditability != "medium" {
		t.Fatalf("short-window grading mismatch: %+v", signals[0])
	}
	if signals[1].Status != "gap" || signals[1].Auditability != "low" {
		t.Fatalf("long-window grading mismatch: %+v", signals[1])
	}
}
