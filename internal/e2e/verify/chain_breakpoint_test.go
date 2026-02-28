package verify

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Clyra-AI/axym/core/normalize"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/proofemit"
	"github.com/Clyra-AI/axym/core/store"
)

func TestChainBreakpoint(t *testing.T) {
	t.Parallel()

	storeDir := filepath.Join(t.TempDir(), "store")
	s, err := store.New(store.Config{RootDir: storeDir, ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}
	emitter := proofemit.Emitter{Store: s, SinkMode: sink.ModeFailClosed}

	first := proofemit.EmitInput{Normalized: normalize.Input{
		SourceType:    "mcp",
		Source:        "mcp-runtime",
		SourceProduct: "axym",
		RecordType:    "tool_invocation",
		Timestamp:     time.Date(2026, 2, 28, 15, 10, 0, 0, time.UTC),
		Event:         map[string]any{"tool_name": "fetch_1"},
		Controls:      normalize.Controls{PermissionsEnforced: true},
	}}
	second := proofemit.EmitInput{Normalized: normalize.Input{
		SourceType:    "mcp",
		Source:        "mcp-runtime",
		SourceProduct: "axym",
		RecordType:    "tool_invocation",
		Timestamp:     time.Date(2026, 2, 28, 15, 11, 0, 0, time.UTC),
		Event:         map[string]any{"tool_name": "fetch_2"},
		Controls:      normalize.Controls{PermissionsEnforced: true},
	}}
	if _, err := emitter.Emit(first); err != nil {
		t.Fatalf("emit first: %v", err)
	}
	if _, err := emitter.Emit(second); err != nil {
		t.Fatalf("emit second: %v", err)
	}

	chain, err := s.LoadChain()
	if err != nil {
		t.Fatalf("LoadChain() error = %v", err)
	}
	chain.Records[0].Event["tool_name"] = "tampered"
	raw, err := json.MarshalIndent(chain, "", "  ")
	if err != nil {
		t.Fatalf("marshal chain: %v", err)
	}
	if err := os.WriteFile(filepath.Join(storeDir, "chain.json"), raw, 0o600); err != nil {
		t.Fatalf("write chain: %v", err)
	}

	stdout, stderr, exit := runAxym(t, "verify", "--chain", "--store-dir", storeDir, "--json")
	if exit != 2 {
		t.Fatalf("exit mismatch: got %d want 2 stdout=%s stderr=%s", exit, stdout, stderr)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode output json: %v", err)
	}
	errObj, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error payload: %s", stdout)
	}
	if errObj["reason"] != "chain_tamper_detected" {
		t.Fatalf("reason mismatch: got %v", errObj["reason"])
	}
	if idx, ok := errObj["break_index"].(float64); !ok || int(idx) != 0 {
		t.Fatalf("break_index mismatch: got %v", errObj["break_index"])
	}
}

func runAxym(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	binaryName := "axym"
	if runtime.GOOS == "windows" {
		binaryName = "axym.exe"
	}
	binaryPath := filepath.Join(t.TempDir(), binaryName)
	build := exec.Command("go", "build", "-o", binaryPath, "./cmd/axym")
	build.Dir = repoRoot
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build axym: %v output=%s", err, string(out))
	}
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), "", 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return string(out), "", exitErr.ExitCode()
	}
	t.Fatalf("run axym: %v", err)
	return "", "", 1
}
