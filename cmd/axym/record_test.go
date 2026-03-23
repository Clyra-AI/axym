package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRecordAddJSONAppendsAndDedupes(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	recordPath := filepath.Join(root, "record.json")
	recordPayload := map[string]any{
		"record_id":      "rec-test-001",
		"source":         "axym",
		"source_product": "axym",
		"agent_id":       "agent-1",
		"record_type":    "decision",
		"timestamp":      "2026-03-01T00:00:00Z",
		"event":          map[string]any{"action": "approve"},
		"metadata":       map[string]any{"ticket": "ABC-1"},
		"controls": map[string]any{
			"permissions_enforced": true,
			"approved_scope":       "default",
		},
	}
	raw, err := json.Marshal(recordPayload)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if err := os.WriteFile(recordPath, raw, 0o600); err != nil {
		t.Fatalf("write record fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"record", "add", "--input", recordPath, "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("first add exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	exit = execute([]string{"record", "add", "--input", recordPath, "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("second add exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v output=%s", err, stdout.String())
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing data envelope: %s", stdout.String())
	}
	if data["deduped"] != true {
		t.Fatalf("expected deduped=true output=%s", stdout.String())
	}
}

func TestRecordAddMissingInputExitCode(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"record", "add", "--json"}, &stdout, &stderr)
	if exit != exitInvalidInput {
		t.Fatalf("exit mismatch: got %d want %d output=%s", exit, exitInvalidInput, stdout.String())
	}
}

func TestRecordAddRejectsUnknownRecordTypeWithSchemaViolationExit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	recordPath := filepath.Join(root, "record.json")
	recordPayload := map[string]any{
		"record_id":      "rec-test-unknown-type",
		"record_version": "v1",
		"source":         "axym",
		"source_product": "axym",
		"agent_id":       "agent-1",
		"record_type":    "does_not_exist",
		"timestamp":      "2026-03-18T00:00:00Z",
		"event":          map[string]any{"anything": true},
		"controls":       map[string]any{"permissions_enforced": true},
	}
	raw, err := json.Marshal(recordPayload)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if err := os.WriteFile(recordPath, raw, 0o600); err != nil {
		t.Fatalf("write record fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"record", "add", "--input", recordPath, "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitPolicyViolation {
		t.Fatalf("exit mismatch: got %d want %d stderr=%s stdout=%s", exit, exitPolicyViolation, stderr.String(), stdout.String())
	}
	if _, err := os.Stat(filepath.Join(storeDir, "chain.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no chain mutation, got err=%v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["reason"] != "schema_violation" {
		t.Fatalf("reason mismatch: got %v output=%s", errObj["reason"], stdout.String())
	}
}

func TestRecordAddRejectsMissingRequiredFieldWithSchemaViolationExit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	recordPath := filepath.Join(root, "record.json")
	recordPayload := map[string]any{
		"record_version": "v1",
		"record_id":      "rec-test-missing-source-product",
		"source":         "manual",
		"agent_id":       "agent-1",
		"record_type":    "decision",
		"timestamp":      "2026-03-18T00:00:00Z",
		"event":          map[string]any{"action": "allow"},
		"controls":       map[string]any{"permissions_enforced": true},
	}
	raw, err := json.Marshal(recordPayload)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if err := os.WriteFile(recordPath, raw, 0o600); err != nil {
		t.Fatalf("write record fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"record", "add", "--input", recordPath, "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitPolicyViolation {
		t.Fatalf("exit mismatch: got %d want %d stderr=%s stdout=%s", exit, exitPolicyViolation, stderr.String(), stdout.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["reason"] != "schema_violation" {
		t.Fatalf("reason mismatch: got %v output=%s", errObj["reason"], stdout.String())
	}
}

func TestRecordAddRejectsWhitespaceOnlyAgentIDWithSchemaViolationExit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	recordPath := filepath.Join(root, "record.json")
	recordPayload := map[string]any{
		"record_version": "v1",
		"record_id":      "rec-test-whitespace-agent",
		"source":         "manual",
		"source_product": "axym",
		"agent_id":       "   ",
		"record_type":    "decision",
		"timestamp":      "2026-03-18T00:00:00Z",
		"event":          map[string]any{"action": "allow"},
		"controls":       map[string]any{"permissions_enforced": true},
	}
	raw, err := json.Marshal(recordPayload)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if err := os.WriteFile(recordPath, raw, 0o600); err != nil {
		t.Fatalf("write record fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"record", "add", "--input", recordPath, "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitPolicyViolation {
		t.Fatalf("exit mismatch: got %d want %d stderr=%s stdout=%s", exit, exitPolicyViolation, stderr.String(), stdout.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["reason"] != "schema_violation" {
		t.Fatalf("reason mismatch: got %v output=%s", errObj["reason"], stdout.String())
	}
}

func TestRecordAddNormalizesLegacyRecordVersionToV1(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	recordPath := filepath.Join(root, "record.json")
	recordPayload := map[string]any{
		"record_version": "1.0",
		"record_id":      "rec-test-legacy-version",
		"source":         "manual",
		"source_product": "axym",
		"agent_id":       "agent-1",
		"record_type":    "decision",
		"timestamp":      "2026-03-18T00:00:00Z",
		"event":          map[string]any{"action": "allow"},
		"controls":       map[string]any{"permissions_enforced": true},
	}
	raw, err := json.Marshal(recordPayload)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if err := os.WriteFile(recordPath, raw, 0o600); err != nil {
		t.Fatalf("write record fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"record", "add", "--input", recordPath, "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("record add exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}

	chainRaw, err := os.ReadFile(filepath.Join(storeDir, "chain.json"))
	if err != nil {
		t.Fatalf("read chain: %v", err)
	}
	var chain map[string]any
	if err := json.Unmarshal(chainRaw, &chain); err != nil {
		t.Fatalf("decode chain: %v", err)
	}
	records, _ := chain["records"].([]any)
	if len(records) != 1 {
		t.Fatalf("expected one record in chain, got %d", len(records))
	}
	record, _ := records[0].(map[string]any)
	if record["record_version"] != "v1" {
		t.Fatalf("record version mismatch: got %v", record["record_version"])
	}
}

func TestRecordAddRejectsUnsupportedRecordVersionWithSchemaViolationExit(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	recordPath := filepath.Join(root, "record.json")
	recordPayload := map[string]any{
		"record_version": "v2",
		"record_id":      "rec-test-unsupported-version",
		"source":         "manual",
		"source_product": "axym",
		"agent_id":       "agent-1",
		"record_type":    "decision",
		"timestamp":      "2026-03-18T00:00:00Z",
		"event":          map[string]any{"action": "allow"},
		"controls":       map[string]any{"permissions_enforced": true},
	}
	raw, err := json.Marshal(recordPayload)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if err := os.WriteFile(recordPath, raw, 0o600); err != nil {
		t.Fatalf("write record fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"record", "add", "--input", recordPath, "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitPolicyViolation {
		t.Fatalf("exit mismatch: got %d want %d stderr=%s stdout=%s", exit, exitPolicyViolation, stderr.String(), stdout.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["reason"] != "schema_violation" {
		t.Fatalf("reason mismatch: got %v output=%s", errObj["reason"], stdout.String())
	}
}

func TestRecordAddRejectsBuiltInSchemaInvalidPayloadWithoutMutation(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	recordPath := filepath.Join(root, "record.json")
	recordPayload := map[string]any{
		"record_id":      "rec-test-bad-dynamic-discovery",
		"record_version": "v1",
		"source":         "axym",
		"source_product": "axym",
		"agent_id":       "agent-1",
		"record_type":    "dynamic_tool_discovery",
		"timestamp":      "2026-03-18T00:00:00Z",
		"event":          map[string]any{},
	}
	raw, err := json.Marshal(recordPayload)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if err := os.WriteFile(recordPath, raw, 0o600); err != nil {
		t.Fatalf("write record fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"record", "add", "--input", recordPath, "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitPolicyViolation {
		t.Fatalf("exit mismatch: got %d want %d stderr=%s stdout=%s", exit, exitPolicyViolation, stderr.String(), stdout.String())
	}
	if _, err := os.Stat(filepath.Join(storeDir, "chain.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no chain mutation, got err=%v", err)
	}
}

func TestRecordAddEmptyMetadataRoundTripsVerify(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	storeDir := filepath.Join(root, "store")
	recordPath := filepath.Join(root, "record.json")
	recordPayload := map[string]any{
		"record_id":      "rec-test-empty-metadata",
		"source":         "axym",
		"source_product": "axym",
		"agent_id":       "agent-1",
		"record_type":    "approval",
		"timestamp":      "2026-03-18T00:00:00Z",
		"event":          map[string]any{"decision": "allow"},
		"metadata":       map[string]any{},
		"controls": map[string]any{
			"permissions_enforced": true,
		},
	}
	raw, err := json.Marshal(recordPayload)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if err := os.WriteFile(recordPath, raw, 0o600); err != nil {
		t.Fatalf("write record fixture: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"record", "add", "--input", recordPath, "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("record add exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	exit = execute([]string{"verify", "--chain", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("verify exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode verify json: %v", err)
	}
	data, _ := payload["data"].(map[string]any)
	verification, _ := data["verification"].(map[string]any)
	if intact, _ := verification["intact"].(bool); !intact {
		t.Fatalf("expected intact chain output=%s", stdout.String())
	}
	if count, _ := verification["count"].(float64); count != 1 {
		t.Fatalf("expected count=1 output=%s", stdout.String())
	}
}
