package gaps

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	corecollect "github.com/Clyra-AI/axym/core/collect"
	"github.com/Clyra-AI/axym/core/collector"
	"github.com/Clyra-AI/axym/core/compliance/coverage"
	"github.com/Clyra-AI/axym/core/compliance/framework"
	"github.com/Clyra-AI/axym/core/compliance/match"
	coregaps "github.com/Clyra-AI/axym/core/gaps"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/store"
)

func TestGapsWorkflowDeterministicRanking(t *testing.T) {
	t.Parallel()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
	fixtureDir := filepath.Join(repoRoot, "fixtures", "collectors")

	req := collector.Request{Now: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC), FixtureDir: fixtureDir}
	registry, err := corecollect.BuildRegistry(req)
	if err != nil {
		t.Fatalf("BuildRegistry: %v", err)
	}
	st, err := store.New(store.Config{RootDir: filepath.Join(t.TempDir(), "store")})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	runner := corecollect.Runner{Registry: registry, Store: st, SinkMode: sink.ModeFailClosed}
	if _, err := runner.Run(context.Background(), req, false); err != nil {
		t.Fatalf("collect runner: %v", err)
	}
	chain, err := st.LoadChain()
	if err != nil {
		t.Fatalf("LoadChain: %v", err)
	}

	frameworks, err := framework.LoadMany([]string{"eu-ai-act", "soc2"})
	if err != nil {
		t.Fatalf("LoadMany: %v", err)
	}
	matchResult := match.Evaluate(frameworks, chain.Records, match.Options{ExcludeInvalidEvidence: true})
	coverageReport := coverage.Build(matchResult)
	first := coregaps.Build(coverageReport)
	second := coregaps.Build(coverageReport)
	left, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first: %v", err)
	}
	right, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second: %v", err)
	}
	if string(left) != string(right) {
		t.Fatalf("expected deterministic gap output")
	}
}
