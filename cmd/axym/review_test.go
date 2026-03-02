package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Clyra-AI/proof"
)

func TestReviewJSONContract(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	seedReviewChain(t, storeDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"review", "--date", "2026-02-28", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if payload["command"] != "review" {
		t.Fatalf("command mismatch: %s", stdout.String())
	}
	data, _ := payload["data"].(map[string]any)
	if got, _ := data["record_count"].(float64); int(got) != 2 {
		t.Fatalf("record count mismatch: %s", stdout.String())
	}
	if _, ok := data["replay_tier_distribution"].([]any); !ok {
		t.Fatalf("missing replay tier distribution: %s", stdout.String())
	}
}

func TestReviewCSVOutput(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	seedReviewChain(t, storeDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"review", "--date", "2026-02-28", "--store-dir", storeDir, "--format", "csv"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	body := stdout.String()
	if !strings.Contains(body, "exception_class,count") {
		t.Fatalf("missing CSV exception header: %s", body)
	}
	if !strings.Contains(body, "record_id,record_type,timestamp,auditability,exception_classes") {
		t.Fatalf("missing CSV record header: %s", body)
	}
}

func TestReviewPDFOutputShape(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	seedReviewChain(t, storeDir)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"review", "--date", "2026-02-28", "--store-dir", storeDir, "--format", "pdf"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	body := stdout.String()
	if !strings.HasPrefix(body, "%PDF-1.4\n") {
		t.Fatalf("missing PDF header: %q", body)
	}
}

func TestReviewInvalidDateExitCode(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"review", "--date", "2026/02/28", "--json"}, &stdout, &stderr)
	if exit != exitInvalidInput {
		t.Fatalf("exit mismatch: got %d stderr=%s stdout=%s", exit, stderr.String(), stdout.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["reason"] != "invalid_input" {
		t.Fatalf("expected invalid_input reason: %s", stdout.String())
	}
}

func seedReviewChain(t *testing.T, storeDir string) {
	t.Helper()
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatalf("mkdir store: %v", err)
	}
	chain := proof.NewChain("review-test")
	replay, err := proof.NewRecord(proof.RecordOpts{
		Source:        "axym",
		SourceProduct: "axym",
		Type:          "replay_certification",
		Timestamp:     mustTime(t, "2026-02-28T10:00:00Z"),
		Event:         map[string]any{"tier": "A", "status": "certified", "pass": true},
		Metadata:      map[string]any{"auditability": "high"},
		Controls:      proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("new replay record: %v", err)
	}
	if err := proof.AppendToChain(chain, replay); err != nil {
		t.Fatalf("append replay: %v", err)
	}
	attach, err := proof.NewRecord(proof.RecordOpts{
		Source:        "axym",
		SourceProduct: "axym",
		Type:          "policy_enforcement",
		Timestamp:     mustTime(t, "2026-02-28T11:00:00Z"),
		Event:         map[string]any{"kind": "ticket_attachment", "status": "dlq", "reason_codes": []any{"TICKET_DLQ"}, "sla_within": false},
		Metadata:      map[string]any{"auditability": "low"},
		Controls:      proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("new attach record: %v", err)
	}
	if err := proof.AppendToChain(chain, attach); err != nil {
		t.Fatalf("append attach: %v", err)
	}
	raw, err := json.MarshalIndent(chain, "", "  ")
	if err != nil {
		t.Fatalf("marshal chain: %v", err)
	}
	if err := os.WriteFile(filepath.Join(storeDir, "chain.json"), raw, 0o600); err != nil {
		t.Fatalf("write chain: %v", err)
	}
}

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}
