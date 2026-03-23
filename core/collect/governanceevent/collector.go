package governanceevent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/collect/collectorerr"
	"github.com/Clyra-AI/axym/core/collector"
	governanceeventschema "github.com/Clyra-AI/axym/schemas/v1/governance_event"
)

const (
	ReasonReadError   = "GOVERNANCE_EVENT_READ_ERROR"
	ReasonSchemaError = "GOVERNANCE_EVENT_SCHEMA_INVALID"
	ReasonParseError  = "GOVERNANCE_EVENT_PARSE_ERROR"
)

type Collector struct{}

func (Collector) Name() string { return "governanceevent" }

func (Collector) Collect(_ context.Context, req collector.Request) (collector.Result, error) {
	if len(req.GovernanceEventFiles) == 0 {
		return collector.Result{ReasonCodes: []string{"NO_INPUT"}}, nil
	}

	candidates := []collector.Candidate{}
	for _, path := range req.GovernanceEventFiles {
		// #nosec G304 -- governance event files are explicit user-provided inputs to collect.
		file, err := os.Open(path)
		if err != nil {
			return collector.Result{}, collectorerr.New(ReasonReadError, "open governance event file", err)
		}
		scanner := bufio.NewScanner(file)
		line := 0
		for scanner.Scan() {
			line++
			raw := bytesTrimSpace(scanner.Bytes())
			if len(raw) == 0 {
				continue
			}
			if err := governanceeventschema.Validate(raw); err != nil {
				_ = file.Close()
				return collector.Result{}, collectorerr.New(ReasonSchemaError, fmt.Sprintf("schema reject at %s:%d", path, line), err)
			}
			var payload map[string]any
			if err := json.Unmarshal(raw, &payload); err != nil {
				_ = file.Close()
				return collector.Result{}, collectorerr.New(ReasonParseError, fmt.Sprintf("decode event at %s:%d", path, line), err)
			}
			candidate, err := promote(payload)
			if err != nil {
				_ = file.Close()
				return collector.Result{}, collectorerr.New(ReasonParseError, fmt.Sprintf("promote event at %s:%d", path, line), err)
			}
			candidates = append(candidates, candidate)
		}
		if err := scanner.Err(); err != nil {
			_ = file.Close()
			return collector.Result{}, collectorerr.New(ReasonReadError, "scan governance event file", err)
		}
		_ = file.Close()
	}
	return collector.Result{Candidates: candidates, ReasonCodes: []string{"CAPTURED"}}, nil
}

func promote(payload map[string]any) (collector.Candidate, error) {
	timestampRaw, _ := payload["timestamp"].(string)
	timestamp, err := time.Parse(time.RFC3339, timestampRaw)
	if err != nil {
		return collector.Candidate{}, fmt.Errorf("parse timestamp: %w", err)
	}
	eventType, _ := payload["event_type"].(string)
	recordType := "decision"
	switch strings.ToLower(eventType) {
	case "approval", "approval_request", "approval_granted":
		recordType = "approval"
	case "policy_eval", "permission_check", "policy_enforcement":
		recordType = "policy_enforcement"
	case "instruction_rewrite", "context_reset", "knowledge_import":
		recordType = "decision"
	}

	actor, _ := payload["actor"].(map[string]any)
	target, _ := payload["target"].(map[string]any)
	source, _ := payload["source"].(string)
	action, _ := payload["action"].(string)
	contextChange, _ := payload["context"].(map[string]any)
	metadata, _ := payload["metadata"].(map[string]any)
	delegationChain, _ := payload["delegation_chain"].([]any)
	actorIdentity := firstString(
		stringFromMap(actor, "id"),
		stringFromMap(payload, "actor_identity"),
	)
	downstreamIdentity := firstString(
		stringFromMap(payload, "downstream_identity"),
		actorIdentity,
	)
	if len(metadata) == 0 {
		metadata = map[string]any{
			"governance_event_type": eventType,
		}
	}

	event := map[string]any{
		"governance_event_type": eventType,
		"governance_source":     source,
		"actor_id":              actor["id"],
		"actor_type":            actor["type"],
		"actor_identity":        actorIdentity,
		"downstream_identity":   downstreamIdentity,
		"action":                action,
		"target_kind":           target["kind"],
		"target_id":             target["id"],
	}
	copyIfPresent(event, "owner_identity", payload, "owner_identity")
	copyIfPresent(event, "policy_digest", payload, "policy_digest")
	copyIfPresent(event, "approval_token_ref", payload, "approval_token_ref")
	if len(delegationChain) > 0 {
		event["delegation_chain"] = delegationChain
	}
	if len(contextChange) > 0 {
		event["context_event_class"] = "context_engineering"
		metadata["context_event_class"] = "context_engineering"
		copyIfPresent(event, "context_previous_hash", contextChange, "previous_hash")
		copyIfPresent(event, "context_current_hash", contextChange, "current_hash")
		copyIfPresent(event, "context_artifact_digest", contextChange, "artifact_digest")
		copyIfPresent(event, "context_artifact_kind", contextChange, "artifact_kind")
		copyIfPresent(event, "context_source_uri", contextChange, "source_uri")
		copyIfPresent(event, "context_reason_code", contextChange, "reason_code")
		copyIfPresent(event, "context_approval_ref", contextChange, "approval_ref")
		if _, ok := event["approval_token_ref"]; !ok {
			copyIfPresent(event, "approval_token_ref", contextChange, "approval_ref")
		}
	}

	return collector.Candidate{
		SourceType:    "governance_event",
		Source:        "governance-event",
		SourceProduct: "axym",
		RecordType:    recordType,
		AgentID:       fmt.Sprintf("%v", actor["id"]),
		Timestamp:     timestamp.UTC().Truncate(time.Second),
		Event:         event,
		Metadata:      metadata,
		Controls:      collector.Controls{PermissionsEnforced: true, ApprovedScope: "governance:event"},
	}, nil
}

func bytesTrimSpace(in []byte) []byte {
	return []byte(strings.TrimSpace(string(in)))
}

func copyIfPresent(dst map[string]any, dstKey string, src map[string]any, srcKey string) {
	if src == nil {
		return
	}
	value, ok := src[srcKey]
	if !ok {
		return
	}
	dst[dstKey] = value
}

func firstString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func stringFromMap(src map[string]any, key string) string {
	if src == nil {
		return ""
	}
	value, _ := src[key].(string)
	return strings.TrimSpace(value)
}
