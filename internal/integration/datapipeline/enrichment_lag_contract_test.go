package datapipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	coreCollect "github.com/Clyra-AI/axym/core/collect"
	"github.com/Clyra-AI/axym/core/collector"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/redact"
	"github.com/Clyra-AI/axym/core/store"
)

func TestEnrichmentLagContract(t *testing.T) {
	t.Parallel()

	fixtureDir := t.TempDir()
	snowflakeFixture := []byte(`{
  "events": [
    {
      "timestamp": "2026-02-28T09:00:07Z",
      "event": {
        "job_name": "daily_models",
        "query_text": "select * from analytics.fact_orders",
        "query_tag": "",
        "enriched_at": "2026-02-28T10:45:00Z"
      },
      "metadata": {
        "evidence_source": "snowflake"
      }
    }
  ]
}`)
	if err := os.WriteFile(filepath.Join(fixtureDir, "snowflake.json"), snowflakeFixture, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	req := collector.Request{Now: time.Date(2026, 2, 28, 9, 0, 0, 0, time.UTC), FixtureDir: fixtureDir}
	registry, err := coreCollect.BuildRegistry(req)
	if err != nil {
		t.Fatalf("BuildRegistry: %v", err)
	}
	evidenceStore, err := store.New(store.Config{RootDir: filepath.Join(t.TempDir(), "store"), ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	runner := coreCollect.Runner{Registry: registry, Store: evidenceStore, SinkMode: sink.ModeFailClosed, Redaction: redact.Config{}}
	result, err := runner.Run(context.Background(), req, false)
	if err != nil {
		t.Fatalf("runner.Run: %v", err)
	}
	if result.Failures != 0 {
		t.Fatalf("unexpected collector failures: %+v", result)
	}

	chain, err := evidenceStore.LoadChain()
	if err != nil {
		t.Fatalf("LoadChain: %v", err)
	}
	found := false
	for _, record := range chain.Records {
		if record.Source != "snowflake" {
			continue
		}
		found = true
		event := record.Event
		decision, _ := event["decision"].(map[string]any)
		if pass, _ := decision["pass"].(bool); pass {
			t.Fatalf("expected decision.pass=false for snowflake event: %+v", event)
		}
		reasons, ok := event["reason_codes"].([]any)
		if !ok {
			raw, _ := json.Marshal(event)
			t.Fatalf("missing reason_codes in snowflake event: %s", raw)
		}
		hasLag := false
		hasMissingTag := false
		for _, reason := range reasons {
			if reason == "ENRICHMENT_LAG" {
				hasLag = true
			}
			if reason == "MISSING_QUERY_TAG" {
				hasMissingTag = true
			}
		}
		if !hasLag || !hasMissingTag {
			t.Fatalf("missing required reason codes: %+v", reasons)
		}
		if _, ok := record.Metadata["replay_inputs"].(map[string]any); !ok {
			t.Fatalf("missing replay_inputs metadata: %+v", record.Metadata)
		}
	}
	if !found {
		t.Fatal("expected at least one snowflake record")
	}
}
