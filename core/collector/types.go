package collector

import (
	"context"
	"time"
)

// Controls carries compliance control hints that are copied into proof records.
type Controls struct {
	PermissionsEnforced bool
	ApprovedScope       string
}

// Candidate is a collector-produced event before normalization/proof emission.
type Candidate struct {
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

// Request configures a single collect cycle.
type Request struct {
	Now                  time.Time
	FixtureDir           string
	PluginCommands       []string
	PluginTimeout        time.Duration
	GovernanceEventFiles []string
}

type Result struct {
	Candidates  []Candidate
	ReasonCodes []string
}

// Collector adapts a source into deterministic Candidate records.
type Collector interface {
	Name() string
	Collect(ctx context.Context, req Request) (Result, error)
}
