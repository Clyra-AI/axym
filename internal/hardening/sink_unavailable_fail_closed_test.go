package hardening

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	corecollect "github.com/Clyra-AI/axym/core/collect"
	"github.com/Clyra-AI/axym/core/collector"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/redact"
	"github.com/Clyra-AI/axym/core/store"
)

func TestHardeningSinkUnavailableFailClosed(t *testing.T) {
	t.Parallel()

	repoRoot := repoRoot(t)
	storeDir := filepath.Join(t.TempDir(), "store")
	if err := os.MkdirAll(filepath.Join(storeDir, "chain.json"), 0o700); err != nil {
		t.Fatalf("mkdir blocking chain path: %v", err)
	}

	req := collector.Request{
		Now:        time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC),
		FixtureDir: filepath.Join(repoRoot, "fixtures", "collectors"),
	}
	registry, err := corecollect.BuildRegistry(req)
	if err != nil {
		t.Fatalf("BuildRegistry: %v", err)
	}
	evidenceStore, err := store.New(store.Config{RootDir: storeDir, ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	runner := corecollect.Runner{
		Registry:  registry,
		Store:     evidenceStore,
		SinkMode:  sink.ModeFailClosed,
		Redaction: redact.Config{},
	}

	_, err = runner.Run(context.Background(), req, false)
	if err == nil {
		t.Fatal("expected fail-closed sink error")
	}
	var collectErr *corecollect.Error
	if !errors.As(err, &collectErr) {
		t.Fatalf("expected *corecollect.Error, got %T", err)
	}
	if collectErr.ReasonCode != "SINK_UNAVAILABLE" {
		t.Fatalf("reason mismatch: got %s", collectErr.ReasonCode)
	}
	info, statErr := os.Stat(filepath.Join(storeDir, "chain.json"))
	if statErr != nil {
		t.Fatalf("stat chain path: %v", statErr)
	}
	if !info.IsDir() {
		t.Fatalf("expected blocking chain path to remain directory, mode=%v", info.Mode())
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
