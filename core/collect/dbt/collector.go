package dbt

import (
	"context"
	"fmt"
	"time"

	"github.com/Clyra-AI/axym/core/collect/collectorerr"
	"github.com/Clyra-AI/axym/core/collect/fixtureutil"
	"github.com/Clyra-AI/axym/core/collector"
	"github.com/Clyra-AI/axym/core/policy/freeze"
	"github.com/Clyra-AI/axym/core/policy/sod"
)

const reasonFixtureError = "DBT_FIXTURE_ERROR"

type Collector struct{}

func (Collector) Name() string { return "dbt" }

func (Collector) Collect(_ context.Context, req collector.Request) (collector.Result, error) {
	events, err := fixtureutil.LoadEvents(req.FixtureDir, "dbt.json")
	if err != nil {
		return collector.Result{}, collectorerr.New(reasonFixtureError, "load dbt fixture", err)
	}
	if len(events) == 0 {
		return collector.Result{ReasonCodes: []string{"NO_INPUT"}}, nil
	}
	fallback := req.Now
	if fallback.IsZero() {
		fallback = time.Date(2026, 2, 28, 0, 5, 0, 0, time.UTC)
	}

	candidates := make([]collector.Candidate, 0, len(events))
	for _, event := range events {
		ts, err := fixtureutil.ParseTimestamp(event.Timestamp, fallback)
		if err != nil {
			return collector.Result{}, collectorerr.New(reasonFixtureError, "parse dbt fixture timestamp", err)
		}
		payload, reasonCodes, err := buildEventPayload(event.Event, ts)
		if err != nil {
			return collector.Result{}, collectorerr.New(reasonFixtureError, "build dbt payload", err)
		}
		metadata := map[string]any{}
		for k, v := range event.Metadata {
			metadata[k] = v
		}
		metadata["replay_inputs"] = map[string]any{
			"job_name":      payload["job_name"],
			"model_digests": payload["model_digests"],
		}
		candidates = append(candidates, collector.Candidate{
			SourceType:    "dbt",
			Source:        "dbt",
			SourceProduct: "axym",
			RecordType:    "data_pipeline_run",
			AgentID:       event.AgentID,
			Timestamp:     ts,
			Event:         payload,
			Metadata:      metadata,
			Controls:      collector.Controls{PermissionsEnforced: true, ApprovedScope: "pipeline:dbt"},
		})
		_ = reasonCodes
	}

	return collector.Result{Candidates: candidates, ReasonCodes: []string{"CAPTURED"}}, nil
}

func buildEventPayload(raw map[string]any, runAt time.Time) (map[string]any, []string, error) {
	jobName, _ := raw["job_name"].(string)
	if jobName == "" {
		return nil, nil, fmt.Errorf("job_name is required")
	}

	requestor, _ := raw["requestor"].(string)
	approver, _ := raw["approver"].(string)
	deployer, _ := raw["deployer"].(string)
	sodDecision := sod.Evaluate(sod.Input{Requestor: requestor, Approver: approver, Deployer: deployer})
	freezeDecision := freeze.Evaluate(runAt, parseFreezeWindows(raw["freeze_windows"]))

	reasonCodes := append([]string{}, sodDecision.ReasonCodes...)
	reasonCodes = append(reasonCodes, freezeDecision.ReasonCodes...)
	pass := sodDecision.Pass && freezeDecision.Pass

	modelDigests := []any{}
	if models, ok := raw["models"].([]any); ok {
		for _, entry := range models {
			model, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			name, _ := model["name"].(string)
			sql, _ := model["sql"].(string)
			if name == "" || sql == "" {
				continue
			}
			modelDigests = append(modelDigests, map[string]any{"name": name, "digest": CanonicalSQLDigest(sql)})
		}
	}

	out := map[string]any{
		"job_name":      jobName,
		"git_sha":       raw["git_sha"],
		"decision":      map[string]any{"pass": pass},
		"reason_codes":  reasonCodes,
		"model_digests": modelDigests,
	}
	return out, reasonCodes, nil
}

func parseFreezeWindows(raw any) []freeze.Window {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	windows := []freeze.Window{}
	for _, item := range items {
		window, ok := item.(map[string]any)
		if !ok {
			continue
		}
		startRaw, _ := window["start"].(string)
		endRaw, _ := window["end"].(string)
		start, startErr := time.Parse(time.RFC3339, startRaw)
		end, endErr := time.Parse(time.RFC3339, endRaw)
		if startErr != nil || endErr != nil {
			continue
		}
		windows = append(windows, freeze.Window{Start: start, End: end})
	}
	return windows
}
