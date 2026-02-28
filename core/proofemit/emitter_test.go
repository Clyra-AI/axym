package proofemit

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/normalize"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/record"
	"github.com/Clyra-AI/axym/core/store"
)

func TestEmitterWritesAndDedupes(t *testing.T) {
	t.Parallel()

	s, err := store.New(store.Config{RootDir: filepath.Join(t.TempDir(), "state"), ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	emitter := Emitter{Store: s, SinkMode: sink.ModeFailClosed}

	input := EmitInput{Normalized: normalize.Input{
		SourceType:    "mcp",
		Source:        "mcp-runtime",
		SourceProduct: "axym",
		RecordType:    "tool_invocation",
		Timestamp:     time.Date(2026, 2, 28, 14, 20, 0, 0, time.UTC),
		Event:         map[string]any{"tool_name": "fetch"},
		Controls:      normalize.Controls{PermissionsEnforced: true},
	}}

	first, err := emitter.Emit(input)
	if err != nil {
		t.Fatalf("first emit error = %v", err)
	}
	second, err := emitter.Emit(input)
	if err != nil {
		t.Fatalf("second emit error = %v", err)
	}
	if !first.Appended {
		t.Fatal("first emit should append")
	}
	if !second.Deduped {
		t.Fatal("second emit should dedupe")
	}
}

func TestChaosSinkFailurePolicies(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "state")
	if _, err := store.New(store.Config{RootDir: root, ComplianceMode: true}); err != nil {
		t.Fatalf("bootstrap store.New() error = %v", err)
	}
	alwaysFail := func(path string, data []byte, fsync bool) error {
		if strings.HasSuffix(path, "chain.json") || strings.HasSuffix(path, "dedupe.json") {
			return errors.New("sink unavailable")
		}
		return store.WriteJSONAtomic(path, data, fsync)
	}
	s, err := store.New(store.Config{RootDir: root, ComplianceMode: true}, store.WithAtomicWriter(alwaysFail))
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}

	input := EmitInput{Normalized: normalize.Input{
		SourceType:    "mcp",
		Source:        "mcp-runtime",
		SourceProduct: "axym",
		RecordType:    "tool_invocation",
		Timestamp:     time.Date(2026, 2, 28, 14, 25, 0, 0, time.UTC),
		Event:         map[string]any{"tool_name": "fetch"},
		Controls:      normalize.Controls{PermissionsEnforced: true},
	}}

	compliance := Emitter{Store: s, SinkMode: sink.ModeFailClosed}
	if _, err := compliance.Emit(input); err == nil {
		t.Fatal("expected compliance mode failure")
	}

	advisory := Emitter{Store: s, SinkMode: sink.ModeAdvisoryOnly}
	result, err := advisory.Emit(input)
	if err != nil {
		t.Fatalf("advisory mode should degrade, got error: %v", err)
	}
	if !result.Degraded || result.ReasonCode != "SINK_UNAVAILABLE" {
		t.Fatalf("expected degraded advisory result, got degraded=%v reason=%q", result.Degraded, result.ReasonCode)
	}
}

func TestEmitRejectsInvalidInputBeforeAppend(t *testing.T) {
	t.Parallel()

	s, err := store.New(store.Config{RootDir: filepath.Join(t.TempDir(), "state"), ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	emitter := Emitter{Store: s, SinkMode: sink.ModeFailClosed}

	_, err = emitter.Emit(EmitInput{Normalized: normalize.Input{
		SourceType: "mcp",
		Event:      map[string]any{"missing": true},
		Controls:   normalize.Controls{PermissionsEnforced: true},
	}})
	if err == nil {
		t.Fatal("expected invalid input error")
	}
	if record.ReasonCode(err) != record.ReasonMappingError {
		t.Fatalf("reason mismatch: got %q", record.ReasonCode(err))
	}

	chain, err := s.LoadChain()
	if err != nil {
		t.Fatalf("LoadChain() error = %v", err)
	}
	if len(chain.Records) != 0 {
		t.Fatalf("invalid input should not append, got %d records", len(chain.Records))
	}
}
