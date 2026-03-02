package stitch

import (
	"testing"
	"time"

	ingeststitch "github.com/Clyra-AI/axym/core/ingest/stitch"
	"github.com/Clyra-AI/proof"
)

func TestHardeningSessionGapDetectionNeverClaimsContinuityWithSyntheticGap(t *testing.T) {
	t.Parallel()

	records := []proof.Record{
		recordAt(t, "session-1", time.Date(2026, 2, 28, 9, 0, 0, 0, time.UTC)),
		recordAt(t, "session-1", time.Date(2026, 2, 28, 9, 1, 0, 0, time.UTC)),
		recordAt(t, "session-2", time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)),
	}

	result := ingeststitch.Analyze(records, ingeststitch.Config{MaxGap: 15 * time.Minute})
	if result.Intact {
		t.Fatalf("synthetic gap must not be reported as intact: %+v", result)
	}
	if len(result.Gaps) == 0 {
		t.Fatalf("expected at least one gap window")
	}
	if result.Gaps[0].ReasonCode != ingeststitch.ReasonChainSessionGap {
		t.Fatalf("unexpected reason code: %+v", result.Gaps[0])
	}
}

func recordAt(t *testing.T, sessionID string, ts time.Time) proof.Record {
	t.Helper()

	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     ts,
		Source:        "wrkr",
		SourceProduct: "wrkr",
		Type:          "scan_finding",
		Event: map[string]any{
			"finding_id": "f-" + ts.Format("150405"),
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
