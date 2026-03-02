package context

import (
	"testing"
	"time"

	"github.com/Clyra-AI/proof"
)

func TestEnrichAndScoreDeterministic(t *testing.T) {
	t.Parallel()

	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		Source:        "mcp",
		SourceProduct: "axym",
		Type:          "tool_invocation",
		Event: map[string]any{
			"tool_name": "filesystem.write",
		},
		Metadata: map[string]any{
			"data_class":       "restricted",
			"risk_level":       "high",
			"discovery_method": "runtime",
		},
		Controls: proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("proof.NewRecord: %v", err)
	}

	ctx := Enrich(*record)
	if ctx.DataClass != "restricted" {
		t.Fatalf("data_class mismatch: %+v", ctx)
	}
	if ctx.EndpointClass != "write" {
		t.Fatalf("endpoint_class mismatch: %+v", ctx)
	}
	if ctx.RiskLevel != "high" {
		t.Fatalf("risk_level mismatch: %+v", ctx)
	}
	if ctx.DiscoveryMethod != "runtime" {
		t.Fatalf("discovery_method mismatch: %+v", ctx)
	}

	weights := Score(ctx)
	if weights.Total != 14 {
		t.Fatalf("weight total mismatch: %+v", weights)
	}
}

func TestEnrichFallsBackToDeterministicDefaults(t *testing.T) {
	t.Parallel()

	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     time.Date(2026, 3, 1, 12, 1, 0, 0, time.UTC),
		Source:        "unknown-source",
		SourceProduct: "axym",
		Type:          "decision",
		Event:         map[string]any{"model": "x"},
		Controls:      proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("proof.NewRecord: %v", err)
	}

	ctx := Enrich(*record)
	if ctx.DataClass != "sensitive" {
		t.Fatalf("expected sensitive fallback, got %+v", ctx)
	}
	if ctx.EndpointClass != "unknown" {
		t.Fatalf("expected unknown endpoint, got %+v", ctx)
	}
	if ctx.RiskLevel != "medium" {
		t.Fatalf("expected medium risk fallback, got %+v", ctx)
	}
	if ctx.DiscoveryMethod != "unknown-source" {
		t.Fatalf("expected source discovery fallback, got %+v", ctx)
	}
}
