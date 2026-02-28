package mcp

import (
	"context"
	"testing"

	"github.com/Clyra-AI/axym/core/collector"
)

func TestCollectNoFixtureProducesNoInput(t *testing.T) {
	t.Parallel()

	result, err := Collector{}.Collect(context.Background(), collector.Request{})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(result.Candidates) != 0 {
		t.Fatalf("expected zero candidates without fixture input: %+v", result.Candidates)
	}
	if len(result.ReasonCodes) != 1 || result.ReasonCodes[0] != "NO_INPUT" {
		t.Fatalf("reason codes mismatch: %+v", result.ReasonCodes)
	}
}
