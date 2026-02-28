package snowflake

import (
	"testing"
	"time"
)

func TestBuildEventPayloadEnrichmentLagAndMissingTag(t *testing.T) {
	t.Parallel()

	runAt := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)
	payload, reasons, err := buildEventPayload(map[string]any{
		"job_name":    "daily_models",
		"query_text":  "SELECT * FROM table",
		"query_tag":   "",
		"enriched_at": "2026-02-28T13:00:01Z",
	}, runAt, 30*time.Minute)
	if err != nil {
		t.Fatalf("buildEventPayload: %v", err)
	}
	if len(reasons) != 2 {
		t.Fatalf("reason mismatch: %+v", reasons)
	}
	decision, _ := payload["decision"].(map[string]any)
	if pass, _ := decision["pass"].(bool); pass {
		t.Fatalf("expected decision.pass=false payload=%+v", payload)
	}
	if _, hasQueryText := payload["query_text"]; hasQueryText {
		t.Fatalf("query_text should not be emitted: %+v", payload)
	}
	if payload["query_digest"] == "" {
		t.Fatalf("query_digest must be present: %+v", payload)
	}
}
