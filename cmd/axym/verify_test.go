package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Clyra-AI/proof"
)

func TestVerifyChainJSONSuccessWithMissingChain(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exit := execute([]string{"verify", "--chain", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitSuccess {
		t.Fatalf("exit mismatch: got %d want %d stderr=%s", exit, exitSuccess, stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if ok, _ := payload["ok"].(bool); !ok {
		t.Fatalf("expected ok=true, payload=%s", stdout.String())
	}
}

func TestVerifyChainTamperExitCodeAndReason(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	if err := os.MkdirAll(storeDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	chain := proof.NewChain("test")
	r1, err := proof.NewRecord(proof.RecordOpts{
		Source:        "axym",
		SourceProduct: "axym",
		Type:          "tool_invocation",
		Timestamp:     time.Date(2026, 2, 28, 15, 0, 0, 0, time.UTC),
		Event:         map[string]any{"tool_name": "fetch"},
		Controls:      proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("NewRecord r1: %v", err)
	}
	if err := proof.AppendToChain(chain, r1); err != nil {
		t.Fatalf("AppendToChain r1: %v", err)
	}
	r2, err := proof.NewRecord(proof.RecordOpts{
		Source:        "axym",
		SourceProduct: "axym",
		Type:          "tool_invocation",
		Timestamp:     time.Date(2026, 2, 28, 15, 1, 0, 0, time.UTC),
		Event:         map[string]any{"tool_name": "fetch2"},
		Controls:      proof.Controls{PermissionsEnforced: true},
	})
	if err != nil {
		t.Fatalf("NewRecord r2: %v", err)
	}
	if err := proof.AppendToChain(chain, r2); err != nil {
		t.Fatalf("AppendToChain r2: %v", err)
	}
	chain.Records[1].Event["tool_name"] = "tampered"
	raw, err := json.MarshalIndent(chain, "", "  ")
	if err != nil {
		t.Fatalf("marshal chain: %v", err)
	}
	if err := os.WriteFile(filepath.Join(storeDir, "chain.json"), raw, 0o600); err != nil {
		t.Fatalf("write chain: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"verify", "--chain", "--store-dir", storeDir, "--json"}, &stdout, &stderr)
	if exit != exitVerificationFailed {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s stderr=%s", exit, exitVerificationFailed, stdout.String(), stderr.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	errObj, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error object: %s", stdout.String())
	}
	if errObj["reason"] != "chain_tamper_detected" {
		t.Fatalf("reason mismatch: got %v", errObj["reason"])
	}
}

func TestVerifyRejectsUnsafeTempPath(t *testing.T) {
	t.Parallel()

	unsafeDir := filepath.Join(t.TempDir(), "unsafe")
	if err := os.MkdirAll(unsafeDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(unsafeDir, "foreign.txt"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write foreign file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := execute([]string{"verify", "--chain", "--store-dir", filepath.Join(t.TempDir(), "store"), "--temp-dir", unsafeDir, "--json"}, &stdout, &stderr)
	if exit != exitUnsafeBlocked {
		t.Fatalf("exit mismatch: got %d want %d stdout=%s stderr=%s", exit, exitUnsafeBlocked, stdout.String(), stderr.String())
	}
}
