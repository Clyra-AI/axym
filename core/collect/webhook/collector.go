package webhook

import (
	"context"
	"time"

	"github.com/Clyra-AI/axym/core/collect/collectorerr"
	"github.com/Clyra-AI/axym/core/collect/fixtureutil"
	"github.com/Clyra-AI/axym/core/collector"
)

const reasonFixtureError = "WEBHOOK_FIXTURE_ERROR"

type Collector struct{}

func (Collector) Name() string { return "webhook" }

func (Collector) Collect(_ context.Context, req collector.Request) (collector.Result, error) {
	events, err := fixtureutil.LoadEvents(req.FixtureDir, "webhook.json")
	if err != nil {
		return collector.Result{}, collectorerr.New(reasonFixtureError, "load webhook fixture", err)
	}
	if len(events) == 0 {
		events = []fixtureutil.Event{{
			Timestamp: "2026-02-28T00:02:00Z",
			Event: map[string]any{
				"webhook_id": "wh-001",
				"event_name": "approval.requested",
			},
			Metadata: map[string]any{
				"evidence_source": "webhook",
				"signature":       "sig-placeholder",
			},
		}}
	}

	fallback := req.Now
	if fallback.IsZero() {
		fallback = time.Date(2026, 2, 28, 0, 2, 0, 0, time.UTC)
	}
	candidates := make([]collector.Candidate, 0, len(events))
	for _, event := range events {
		ts, err := fixtureutil.ParseTimestamp(event.Timestamp, fallback)
		if err != nil {
			return collector.Result{}, collectorerr.New(reasonFixtureError, "parse webhook fixture timestamp", err)
		}
		candidates = append(candidates, collector.Candidate{
			SourceType:    "webhook",
			Source:        "webhook-ingest",
			SourceProduct: "axym",
			RecordType:    "policy_enforcement",
			AgentID:       event.AgentID,
			Timestamp:     ts,
			Event:         event.Event,
			Metadata:      event.Metadata,
			Controls:      collector.Controls{PermissionsEnforced: true, ApprovedScope: "governance:event"},
		})
	}
	return collector.Result{Candidates: candidates, ReasonCodes: []string{"CAPTURED"}}, nil
}
