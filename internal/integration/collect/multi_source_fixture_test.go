package collect

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	coreCollect "github.com/Clyra-AI/axym/core/collect"
	"github.com/Clyra-AI/axym/core/collector"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/redact"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/proof"
)

func TestMultiSourceFixtureCaptureAndSchemaParity(t *testing.T) {
	t.Parallel()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	fixtureDir := filepath.Join(repoRoot, "fixtures", "collectors")

	req := collector.Request{
		Now:        time.Date(2026, 2, 28, 9, 0, 0, 0, time.UTC),
		FixtureDir: fixtureDir,
	}
	registry, err := coreCollect.BuildRegistry(req)
	if err != nil {
		t.Fatalf("BuildRegistry: %v", err)
	}

	evidenceStore, err := store.New(store.Config{RootDir: filepath.Join(t.TempDir(), "store"), ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	runner := coreCollect.Runner{
		Registry: registry,
		Store:    evidenceStore,
		SinkMode: sink.ModeFailClosed,
		Redaction: redact.Config{
			MetadataRules: []redact.Rule{{Path: "auth_token", Action: redact.ActionHash}, {Path: "api_key", Action: redact.ActionHash}},
		},
	}
	result, err := runner.Run(context.Background(), req, false)
	if err != nil {
		t.Fatalf("runner.Run: %v", err)
	}
	if result.Failures != 0 {
		t.Fatalf("unexpected failures: %+v", result)
	}
	if result.Captured != 7 {
		t.Fatalf("captured mismatch: got %d want 7", result.Captured)
	}
	if result.Appended != 7 {
		t.Fatalf("appended mismatch: got %d want 7", result.Appended)
	}

	chain, err := evidenceStore.LoadChain()
	if err != nil {
		t.Fatalf("LoadChain: %v", err)
	}
	if len(chain.Records) != 7 {
		t.Fatalf("record count mismatch: got %d want 7", len(chain.Records))
	}
	verification, err := proof.VerifyChain(chain)
	if err != nil {
		t.Fatalf("proof.VerifyChain: %v", err)
	}
	if !verification.Intact {
		t.Fatalf("expected intact chain: %+v", verification)
	}

	var sawHashedToken bool
	for i := range chain.Records {
		record := chain.Records[i]
		if err := proof.ValidateRecord(&record); err != nil {
			t.Fatalf("invalid record at %d: %v", i, err)
		}
		if value, ok := record.Metadata["auth_token"].(string); ok {
			sawHashedToken = strings.HasPrefix(value, "sha256:")
		}
	}
	if !sawHashedToken {
		t.Fatal("expected redacted hashed token in mcp metadata")
	}
}
