package normalize

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Input struct {
	SourceType    string
	Source        string
	SourceProduct string
	RecordType    string
	AgentID       string
	Timestamp     time.Time
	Event         map[string]any
	Metadata      map[string]any
	Controls      Controls
}

type Controls struct {
	PermissionsEnforced bool   `json:"permissions_enforced"`
	ApprovedScope       string `json:"approved_scope,omitempty"`
}

type Record struct {
	Source        string
	SourceProduct string
	RecordType    string
	AgentID       string
	Timestamp     time.Time
	Event         map[string]any
	Metadata      map[string]any
	Controls      Controls
}

type sourceContract struct {
	DefaultRecordType string
	RequiredEventKeys []string
}

var contracts = map[string]sourceContract{
	"mcp": {
		DefaultRecordType: "tool_invocation",
		RequiredEventKeys: []string{"tool_name"},
	},
	"llmapi": {
		DefaultRecordType: "decision",
		RequiredEventKeys: []string{"model"},
	},
	"cicd": {
		DefaultRecordType: "deployment",
		RequiredEventKeys: []string{"pipeline"},
	},
	"githubactions": {
		DefaultRecordType: "deployment",
		RequiredEventKeys: []string{"workflow"},
	},
	"data_pipeline": {
		DefaultRecordType: "data_pipeline_run",
		RequiredEventKeys: []string{"job_name"},
	},
	"dbt": {
		DefaultRecordType: "data_pipeline_run",
		RequiredEventKeys: []string{"job_name"},
	},
	"snowflake": {
		DefaultRecordType: "data_pipeline_run",
		RequiredEventKeys: []string{"job_name"},
	},
	"webhook": {
		DefaultRecordType: "policy_enforcement",
		RequiredEventKeys: []string{"webhook_id"},
	},
	"gitmeta": {
		DefaultRecordType: "model_change",
		RequiredEventKeys: []string{"commit_sha"},
	},
	"governance_event": {
		DefaultRecordType: "policy_enforcement",
		RequiredEventKeys: []string{"governance_event_type"},
	},
	"plugin": {
		DefaultRecordType: "tool_invocation",
		RequiredEventKeys: []string{},
	},
	"wrkr": {
		DefaultRecordType: "scan_finding",
		RequiredEventKeys: []string{"finding_id"},
	},
	"gait": {
		DefaultRecordType: "policy_enforcement",
		RequiredEventKeys: []string{"decision"},
	},
}

func Normalize(in Input) (Record, error) {
	sourceType := strings.ToLower(strings.TrimSpace(in.SourceType))
	contract, ok := contracts[sourceType]
	if !ok {
		return Record{}, fmt.Errorf("unsupported source_type %q", in.SourceType)
	}
	if in.Event == nil {
		return Record{}, fmt.Errorf("event is required")
	}

	event, err := normalizeObject(in.Event)
	if err != nil {
		return Record{}, fmt.Errorf("normalize event: %w", err)
	}
	metadata, err := normalizeObject(in.Metadata)
	if err != nil {
		return Record{}, fmt.Errorf("normalize metadata: %w", err)
	}
	applyAliases(sourceType, event)
	for _, key := range contract.RequiredEventKeys {
		if _, exists := event[key]; !exists {
			return Record{}, fmt.Errorf("missing required event key %q for source_type %q", key, sourceType)
		}
	}

	source := strings.TrimSpace(in.Source)
	if source == "" {
		source = sourceType
	}
	sourceProduct := strings.TrimSpace(in.SourceProduct)
	if sourceProduct == "" {
		sourceProduct = "axym"
	}
	recordType := strings.TrimSpace(in.RecordType)
	if recordType == "" {
		recordType = contract.DefaultRecordType
	}
	timestamp := in.Timestamp.UTC().Truncate(time.Second)
	if timestamp.IsZero() {
		timestamp = time.Now().UTC().Truncate(time.Second)
	}

	return Record{
		Source:        source,
		SourceProduct: sourceProduct,
		RecordType:    recordType,
		AgentID:       strings.TrimSpace(in.AgentID),
		Timestamp:     timestamp,
		Event:         event,
		Metadata:      metadata,
		Controls:      in.Controls,
	}, nil
}

func normalizeObject(in map[string]any) (map[string]any, error) {
	if in == nil {
		return map[string]any{}, nil
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		return map[string]any{}, nil
	}
	return out, nil
}

func applyAliases(sourceType string, event map[string]any) {
	switch sourceType {
	case "mcp":
		if _, ok := event["tool_name"]; !ok {
			if tool, exists := event["tool"]; exists {
				event["tool_name"] = tool
			}
		}
	case "data_pipeline":
		if _, ok := event["job_name"]; !ok {
			if job, exists := event["job"]; exists {
				event["job_name"] = job
			}
		}
	case "cicd":
		if _, ok := event["pipeline"]; !ok {
			if p, exists := event["pipeline_name"]; exists {
				event["pipeline"] = p
			}
		}
	}
}
