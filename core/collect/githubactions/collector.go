package githubactions

import (
	"context"
	"time"

	"github.com/Clyra-AI/axym/core/collect/collectorerr"
	"github.com/Clyra-AI/axym/core/collect/fixtureutil"
	"github.com/Clyra-AI/axym/core/collector"
)

const reasonFixtureError = "GITHUB_ACTIONS_FIXTURE_ERROR"

type Collector struct{}

func (Collector) Name() string { return "githubactions" }

func (Collector) Collect(_ context.Context, req collector.Request) (collector.Result, error) {
	events, err := fixtureutil.LoadEvents(req.FixtureDir, "githubactions.json")
	if err != nil {
		return collector.Result{}, collectorerr.New(reasonFixtureError, "load githubactions fixture", err)
	}
	if len(events) == 0 {
		events = []fixtureutil.Event{{
			Timestamp: "2026-02-28T00:03:00Z",
			Event: map[string]any{
				"workflow": "deploy",
				"status":   "success",
			},
			Metadata: map[string]any{
				"evidence_source": "githubactions",
				"run_id":          "1001",
			},
		}}
	}

	fallback := req.Now
	if fallback.IsZero() {
		fallback = time.Date(2026, 2, 28, 0, 3, 0, 0, time.UTC)
	}
	candidates := make([]collector.Candidate, 0, len(events))
	for _, event := range events {
		ts, err := fixtureutil.ParseTimestamp(event.Timestamp, fallback)
		if err != nil {
			return collector.Result{}, collectorerr.New(reasonFixtureError, "parse githubactions fixture timestamp", err)
		}
		candidates = append(candidates, collector.Candidate{
			SourceType:    "githubactions",
			Source:        "github-actions",
			SourceProduct: "axym",
			RecordType:    "deployment",
			AgentID:       event.AgentID,
			Timestamp:     ts,
			Event:         event.Event,
			Metadata:      event.Metadata,
			Controls:      collector.Controls{PermissionsEnforced: true, ApprovedScope: "pipeline:deploy"},
		})
	}
	return collector.Result{Candidates: candidates, ReasonCodes: []string{"CAPTURED"}}, nil
}
