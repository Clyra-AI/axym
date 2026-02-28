package llmapi

import (
	"context"
	"time"

	"github.com/Clyra-AI/axym/core/collect/collectorerr"
	"github.com/Clyra-AI/axym/core/collect/fixtureutil"
	"github.com/Clyra-AI/axym/core/collector"
)

const reasonFixtureError = "LLMAPI_FIXTURE_ERROR"

type Collector struct{}

func (Collector) Name() string { return "llmapi" }

func (Collector) Collect(_ context.Context, req collector.Request) (collector.Result, error) {
	events, err := fixtureutil.LoadEvents(req.FixtureDir, "llmapi.json")
	if err != nil {
		return collector.Result{}, collectorerr.New(reasonFixtureError, "load llmapi fixture", err)
	}
	if len(events) == 0 {
		events = []fixtureutil.Event{{
			Timestamp: "2026-02-28T00:01:00Z",
			Event: map[string]any{
				"model":    "gpt-4.1",
				"decision": "allow",
			},
			Metadata: map[string]any{
				"evidence_source": "llmapi",
				"api_key":         "secret-key-placeholder",
			},
		}}
	}

	fallback := req.Now
	if fallback.IsZero() {
		fallback = time.Date(2026, 2, 28, 0, 1, 0, 0, time.UTC)
	}
	candidates := make([]collector.Candidate, 0, len(events))
	for _, event := range events {
		ts, err := fixtureutil.ParseTimestamp(event.Timestamp, fallback)
		if err != nil {
			return collector.Result{}, collectorerr.New(reasonFixtureError, "parse llmapi fixture timestamp", err)
		}
		candidates = append(candidates, collector.Candidate{
			SourceType:    "llmapi",
			Source:        "llm-middleware",
			SourceProduct: "axym",
			RecordType:    "decision",
			AgentID:       event.AgentID,
			Timestamp:     ts,
			Event:         event.Event,
			Metadata:      event.Metadata,
			Controls:      collector.Controls{PermissionsEnforced: true, ApprovedScope: "model:invoke"},
		})
	}
	return collector.Result{Candidates: candidates, ReasonCodes: []string{"CAPTURED"}}, nil
}
