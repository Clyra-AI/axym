package dbt

import (
	"testing"
	"time"
)

func TestBuildEventPayloadDigestOnlyAndPolicySemantics(t *testing.T) {
	t.Parallel()

	runAt := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)
	payload, reasonCodes, err := buildEventPayload(map[string]any{
		"job_name":  "daily_models",
		"requestor": "alice",
		"approver":  "bob",
		"deployer":  "alice",
		"models": []any{
			map[string]any{"name": "fact", "sql": "SELECT * FROM raw.table"},
		},
		"freeze_windows": []any{
			map[string]any{"start": "2026-02-28T11:00:00Z", "end": "2026-02-28T13:00:00Z"},
		},
	}, runAt)
	if err != nil {
		t.Fatalf("buildEventPayload: %v", err)
	}
	decision, ok := payload["decision"].(map[string]any)
	if !ok {
		t.Fatalf("missing decision payload: %+v", payload)
	}
	if pass, _ := decision["pass"].(bool); pass {
		t.Fatalf("expected decision.pass=false payload=%+v", payload)
	}
	if len(reasonCodes) < 2 {
		t.Fatalf("expected SoD + freeze reasons, got %+v", reasonCodes)
	}
	modelDigests, ok := payload["model_digests"].([]any)
	if !ok || len(modelDigests) != 1 {
		t.Fatalf("model digests mismatch: %+v", payload["model_digests"])
	}
	digestMap, _ := modelDigests[0].(map[string]any)
	if _, hasSQL := digestMap["sql"]; hasSQL {
		t.Fatalf("raw sql should not be emitted: %+v", digestMap)
	}
	if digest, _ := digestMap["digest"].(string); digest == "" {
		t.Fatalf("digest missing: %+v", digestMap)
	}
}
