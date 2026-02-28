package snowflake

import (
	"context"
	"fmt"
	"time"

	"github.com/Clyra-AI/axym/core/collect/collectorerr"
	"github.com/Clyra-AI/axym/core/collect/fixtureutil"
	"github.com/Clyra-AI/axym/core/collector"
)

const (
	reasonFixtureError  = "SNOWFLAKE_FIXTURE_ERROR"
	ReasonEnrichmentLag = "ENRICHMENT_LAG"
	ReasonMissingTag    = "MISSING_QUERY_TAG"
)

type Collector struct {
	MaxEnrichmentLag time.Duration
}

func (Collector) Name() string { return "snowflake" }

func (c Collector) Collect(_ context.Context, req collector.Request) (collector.Result, error) {
	events, err := fixtureutil.LoadEvents(req.FixtureDir, "snowflake.json")
	if err != nil {
		return collector.Result{}, collectorerr.New(reasonFixtureError, "load snowflake fixture", err)
	}
	if len(events) == 0 {
		events = []fixtureutil.Event{{
			Timestamp: "2026-02-28T00:06:00Z",
			Event: map[string]any{
				"job_name":       "daily_models",
				"query_text":     "select count(*) from analytics.fact_orders",
				"query_tag":      "CHANGE-1234",
				"enriched_at":    "2026-02-28T00:10:00Z",
				"requestor":      "alice",
				"approver":       "bob",
				"deployer":       "carol",
				"warehouse_name": "COMPLIANCE_WH",
			},
			Metadata: map[string]any{"evidence_source": "snowflake"},
		}}
	}
	maxLag := c.MaxEnrichmentLag
	if maxLag <= 0 {
		maxLag = 30 * time.Minute
	}

	fallback := req.Now
	if fallback.IsZero() {
		fallback = time.Date(2026, 2, 28, 0, 6, 0, 0, time.UTC)
	}

	candidates := make([]collector.Candidate, 0, len(events))
	for _, event := range events {
		ts, err := fixtureutil.ParseTimestamp(event.Timestamp, fallback)
		if err != nil {
			return collector.Result{}, collectorerr.New(reasonFixtureError, "parse snowflake fixture timestamp", err)
		}
		payload, reasons, err := buildEventPayload(event.Event, ts, maxLag)
		if err != nil {
			return collector.Result{}, collectorerr.New(reasonFixtureError, "build snowflake payload", err)
		}
		metadata := map[string]any{}
		for k, v := range event.Metadata {
			metadata[k] = v
		}
		metadata["replay_inputs"] = map[string]any{
			"job_name":     payload["job_name"],
			"query_digest": payload["query_digest"],
		}
		metadata["reason_codes"] = reasons

		candidates = append(candidates, collector.Candidate{
			SourceType:    "snowflake",
			Source:        "snowflake",
			SourceProduct: "axym",
			RecordType:    "data_pipeline_run",
			AgentID:       event.AgentID,
			Timestamp:     ts,
			Event:         payload,
			Metadata:      metadata,
			Controls:      collector.Controls{PermissionsEnforced: true, ApprovedScope: "pipeline:snowflake"},
		})
	}
	return collector.Result{Candidates: candidates, ReasonCodes: []string{"CAPTURED"}}, nil
}

func buildEventPayload(raw map[string]any, runAt time.Time, maxLag time.Duration) (map[string]any, []string, error) {
	jobName, _ := raw["job_name"].(string)
	if jobName == "" {
		return nil, nil, fmt.Errorf("job_name is required")
	}
	queryText, _ := raw["query_text"].(string)
	if queryText == "" {
		return nil, nil, fmt.Errorf("query_text is required")
	}
	queryTag, _ := raw["query_tag"].(string)
	reasons := []string{}
	if queryTag == "" {
		reasons = append(reasons, ReasonMissingTag)
	}
	if enrichedAtRaw, _ := raw["enriched_at"].(string); enrichedAtRaw != "" {
		enrichedAt, err := time.Parse(time.RFC3339, enrichedAtRaw)
		if err != nil {
			return nil, nil, fmt.Errorf("parse enriched_at: %w", err)
		}
		if enrichedAt.UTC().Sub(runAt.UTC()) > maxLag {
			reasons = append(reasons, ReasonEnrichmentLag)
		}
	}
	pass := len(reasons) == 0
	out := map[string]any{
		"job_name":       jobName,
		"warehouse_name": raw["warehouse_name"],
		"query_digest":   CanonicalQueryDigest(queryText),
		"query_tag":      queryTag,
		"decision":       map[string]any{"pass": pass},
		"reason_codes":   reasons,
	}
	return out, reasons, nil
}
