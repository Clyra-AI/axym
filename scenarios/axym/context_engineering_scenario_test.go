package axym

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	corebundle "github.com/Clyra-AI/axym/core/bundle"
	corecollect "github.com/Clyra-AI/axym/core/collect"
	"github.com/Clyra-AI/axym/core/collector"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/store"
	verifybundle "github.com/Clyra-AI/axym/core/verify/bundle"
)

func TestScenarioContextEngineeringEvidenceFlow(t *testing.T) {
	t.Parallel()

	repoRoot := scenarioRepoRoot(t)
	governancePath := filepath.Join(repoRoot, "fixtures", "governance", "context_engineering.jsonl")

	req := collector.Request{
		Now:                  time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC),
		GovernanceEventFiles: []string{governancePath},
	}
	registry, err := corecollect.BuildRegistry(req)
	if err != nil {
		t.Fatalf("BuildRegistry: %v", err)
	}
	storeDir := filepath.Join(t.TempDir(), "store")
	st, err := store.New(store.Config{RootDir: storeDir, ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	runner := corecollect.Runner{Registry: registry, Store: st, SinkMode: sink.ModeFailClosed}
	result, err := runner.Run(context.Background(), req, false)
	if err != nil {
		t.Fatalf("runner.Run: %v", err)
	}
	if result.Captured != 3 || result.Appended != 3 {
		t.Fatalf("unexpected collect result: %+v", result)
	}

	bundleDir := filepath.Join(t.TempDir(), "bundle")
	bundleResult, err := corebundle.Build(corebundle.BuildRequest{
		AuditName:    "Q3-2026",
		FrameworkIDs: []string{"eu-ai-act", "soc2"},
		StoreDir:     storeDir,
		OutputDir:    bundleDir,
	})
	if err != nil {
		t.Fatalf("bundle.Build: %v", err)
	}
	verifyResult, err := verifybundle.Verify(bundleResult.Path, []string{"eu-ai-act", "soc2"})
	if err != nil {
		t.Fatalf("verifybundle.Verify: %v", err)
	}
	if !verifyResult.Cryptographic || !verifyResult.ComplianceVerified {
		t.Fatalf("unexpected verify result: %+v", verifyResult)
	}

	rawRecords, err := os.ReadFile(filepath.Join(bundleResult.Path, "raw-records.jsonl"))
	if err != nil {
		t.Fatalf("read raw-records: %v", err)
	}
	if !strings.Contains(string(rawRecords), `"context_event_class":"context_engineering"`) {
		t.Fatalf("raw records missing context engineering evidence: %s", string(rawRecords))
	}
}

func scenarioRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
