package datapipeline

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/collect/snowflake"
	"github.com/Clyra-AI/axym/core/collector"
)

func TestHardeningUndecidablePathDoesNotPass(t *testing.T) {
	t.Parallel()

	fixtureDir := t.TempDir()
	raw := []byte(`{
  "events": [
    {
      "timestamp": "2026-02-28T12:00:00Z",
      "event": {
        "job_name": "daily_models",
        "query_text": "select * from t",
        "query_tag": "",
        "enriched_at": "2026-02-28T13:00:01Z"
      }
    }
  ]
}`)
	if err := os.WriteFile(filepath.Join(fixtureDir, "snowflake.json"), raw, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	result, err := snowflake.Collector{MaxEnrichmentLag: 30 * time.Minute}.Collect(context.Background(), collector.Request{FixtureDir: fixtureDir})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidate mismatch: %+v", result)
	}
	decision, _ := result.Candidates[0].Event["decision"].(map[string]any)
	if pass, _ := decision["pass"].(bool); pass {
		payload, _ := json.Marshal(result.Candidates[0].Event)
		t.Fatalf("undecidable high-risk path must not pass: %s", payload)
	}
}
