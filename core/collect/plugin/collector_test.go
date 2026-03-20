package plugin

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/collector"
	"github.com/Clyra-AI/proof"
)

func TestMain(m *testing.M) {
	switch os.Getenv("AXYM_PLUGIN_MODE") {
	case "malformed":
		fmt.Println("{not-json")
		os.Exit(0)
	case "timeout":
		time.Sleep(2 * time.Second)
		os.Exit(0)
	case "valid":
		fmt.Println(`{"source_type":"plugin","source":"custom","source_product":"axym","record_type":"tool_invocation","timestamp":"2026-02-28T12:00:00Z","event":{"tool_name":"custom_tool"},"metadata":{"evidence_source":"plugin"},"controls":{"permissions_enforced":true}}`)
		os.Exit(0)
	case "relationship":
		fmt.Println(`{"source_type":"plugin","source":"custom","source_product":"axym","record_type":"tool_invocation","timestamp":"2026-02-28T12:00:00Z","event":{"tool_name":"custom_tool"},"metadata":{"evidence_source":"plugin"},"relationship":{"parent_ref":{"kind":"trace","id":"trace-123"},"entity_refs":[{"kind":"resource","id":"db://prod"}],"policy_ref":{"policy_digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},"controls":{"permissions_enforced":true}}`)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestMalformedJSONLRejected(t *testing.T) {
	t.Parallel()

	pluginCollector := Collector{Command: helperCommand(t), Timeout: time.Second, Env: []string{"AXYM_PLUGIN_MODE=malformed"}}
	_, err := pluginCollector.Collect(context.Background(), collector.Request{})
	if err == nil {
		t.Fatal("expected malformed plugin output error")
	}
	if rc, ok := err.(interface{ ReasonCode() string }); !ok || rc.ReasonCode() != ReasonMalformed {
		t.Fatalf("reason mismatch: err=%v", err)
	}
}

func TestChaosPluginTimeoutReasonClassification(t *testing.T) {
	t.Parallel()

	pluginCollector := Collector{Command: helperCommand(t), Timeout: 50 * time.Millisecond, Env: []string{"AXYM_PLUGIN_MODE=timeout"}}
	_, err := pluginCollector.Collect(context.Background(), collector.Request{})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if rc, ok := err.(interface{ ReasonCode() string }); !ok || rc.ReasonCode() != ReasonTimeout {
		t.Fatalf("reason mismatch: err=%v", err)
	}
}

func TestValidPluginOutputAccepted(t *testing.T) {
	t.Parallel()

	pluginCollector := Collector{Command: helperCommand(t), Timeout: time.Second, Env: []string{"AXYM_PLUGIN_MODE=valid"}}
	result, err := pluginCollector.Collect(context.Background(), collector.Request{})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidate count mismatch: %+v", result)
	}
	if result.Candidates[0].Event["tool_name"] != "custom_tool" {
		t.Fatalf("event mismatch: %+v", result.Candidates[0].Event)
	}
}

func TestPluginRelationshipOutputAccepted(t *testing.T) {
	t.Parallel()

	pluginCollector := Collector{Command: helperCommand(t), Timeout: time.Second, Env: []string{"AXYM_PLUGIN_MODE=relationship"}}
	result, err := pluginCollector.Collect(context.Background(), collector.Request{})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidate count mismatch: %+v", result)
	}
	relationship := result.Candidates[0].Relationship
	if relationship == nil {
		t.Fatal("expected relationship to be preserved")
	}
	if relationship.ParentRef == nil || relationship.ParentRef.Kind != "trace" || relationship.ParentRef.ID != "trace-123" {
		t.Fatalf("parent ref mismatch: %+v", relationship)
	}
	foundResource := false
	for _, ref := range relationship.EntityRefs {
		if ref.Kind == "resource" && ref.ID == "db://prod" {
			foundResource = true
			break
		}
	}
	if !foundResource {
		t.Fatalf("expected resource ref in relationship: %+v", relationship.EntityRefs)
	}
	if relationship.PolicyRef == nil || relationship.PolicyRef.PolicyDigest != "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("policy ref mismatch: %+v", relationship.PolicyRef)
	}
	if _, ok := any(relationship).(*proof.Relationship); !ok {
		t.Fatalf("relationship type mismatch: %T", relationship)
	}
}

func helperCommand(t *testing.T) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	return exe
}
