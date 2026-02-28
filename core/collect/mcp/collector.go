package mcp

import (
	"context"
	"time"

	"github.com/Clyra-AI/axym/core/collect/collectorerr"
	"github.com/Clyra-AI/axym/core/collect/fixtureutil"
	"github.com/Clyra-AI/axym/core/collector"
)

const reasonFixtureError = "MCP_FIXTURE_ERROR"

type Collector struct{}

func (Collector) Name() string { return "mcp" }

func (Collector) Collect(_ context.Context, req collector.Request) (collector.Result, error) {
	events, err := fixtureutil.LoadEvents(req.FixtureDir, "mcp.json")
	if err != nil {
		return collector.Result{}, collectorerr.New(reasonFixtureError, "load mcp fixture", err)
	}
	if len(events) == 0 {
		return collector.Result{ReasonCodes: []string{"NO_INPUT"}}, nil
	}

	candidates := make([]collector.Candidate, 0, len(events))
	fallback := req.Now
	if fallback.IsZero() {
		fallback = time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
	}
	for _, event := range events {
		ts, err := fixtureutil.ParseTimestamp(event.Timestamp, fallback)
		if err != nil {
			return collector.Result{}, collectorerr.New(reasonFixtureError, "parse mcp fixture timestamp", err)
		}
		candidates = append(candidates, collector.Candidate{
			SourceType:    "mcp",
			Source:        "axym-mcp-collector",
			SourceProduct: "axym",
			RecordType:    "tool_invocation",
			AgentID:       event.AgentID,
			Timestamp:     ts,
			Event:         event.Event,
			Metadata:      event.Metadata,
			Controls:      collector.Controls{PermissionsEnforced: true, ApprovedScope: "tool:read"},
		})
	}
	return collector.Result{Candidates: candidates, ReasonCodes: []string{"CAPTURED"}}, nil
}
