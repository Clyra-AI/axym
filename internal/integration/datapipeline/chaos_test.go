package datapipeline

import (
	"context"
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

func TestChaosCollectorMalformedPayloadDoesNotBlockOtherCollectors(t *testing.T) {
	t.Parallel()

	fixtureDir := t.TempDir()
	badSnowflake := []byte(`{"events":[{"timestamp":"bad-time","event":{"job_name":"daily_models","query_text":"select 1"}}]}`)
	if err := os.WriteFile(filepath.Join(fixtureDir, "snowflake.json"), badSnowflake, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	mcpFixture := []byte(`{"events":[{"timestamp":"2026-02-28T09:00:01Z","event":{"tool_name":"filesystem.read","action":"read"},"metadata":{"evidence_source":"mcp"}}]}`)
	if err := os.WriteFile(filepath.Join(fixtureDir, "mcp.json"), mcpFixture, 0o600); err != nil {
		t.Fatalf("write mcp fixture: %v", err)
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
	if result.Failures == 0 {
		t.Fatalf("expected at least one collector failure: %+v", result)
	}
	if result.Captured == 0 {
		t.Fatalf("expected other collectors to continue capturing: %+v", result)
	}
}
