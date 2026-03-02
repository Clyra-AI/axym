package match

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	corecollect "github.com/Clyra-AI/axym/core/collect"
	"github.com/Clyra-AI/axym/core/collector"
	"github.com/Clyra-AI/axym/core/compliance/framework"
	"github.com/Clyra-AI/axym/core/compliance/match"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/store"
)

func TestMapWorkflowFixtureDeterministic(t *testing.T) {
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
	result := match.Evaluate(frameworks, chain.Records, match.Options{ExcludeInvalidEvidence: true})
	if result.Summary.ControlCount == 0 {
		t.Fatalf("expected control coverage result: %+v", result.Summary)
	}
	if result.Summary.CoveredCount == 0 {
		t.Fatalf("expected at least one covered control from fixture corpus: %+v", result.Summary)
	}
}
