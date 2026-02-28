package record

import (
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/normalize"
	"github.com/Clyra-AI/axym/core/record"
	"github.com/Clyra-AI/axym/core/redact"
	"github.com/Clyra-AI/proof"
)

func TestNormalizeValidateIntegration(t *testing.T) {
	t.Parallel()

	r, err := record.NormalizeAndBuild(normalize.Input{
		SourceType:    "mcp",
		Source:        "mcp-runtime",
		SourceProduct: "axym",
		RecordType:    "tool_invocation",
		Timestamp:     time.Date(2026, 2, 28, 12, 30, 0, 0, time.UTC),
		Event: map[string]any{
			"tool_name": "fetch_url",
			"status":    "success",
		},
		Controls: normalize.Controls{PermissionsEnforced: true},
	}, redact.Config{})
	if err != nil {
		t.Fatalf("NormalizeAndBuild() error = %v", err)
	}
	if err := proof.ValidateRecord(r); err != nil {
		t.Fatalf("proof.ValidateRecord() error = %v", err)
	}
}

func TestNormalizeValidateRejectsMissingRequiredSourceField(t *testing.T) {
	t.Parallel()

	_, err := record.NormalizeAndBuild(normalize.Input{
		SourceType: "mcp",
		Event:      map[string]any{"status": "ok"},
		Controls:   normalize.Controls{PermissionsEnforced: true},
	}, redact.Config{})
	if err == nil {
		t.Fatal("expected error for missing source-specific key")
	}
	if record.ReasonCode(err) != record.ReasonMappingError {
		t.Fatalf("reason mismatch: got %q", record.ReasonCode(err))
	}
}
