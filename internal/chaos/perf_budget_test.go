package chaos

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	corebundle "github.com/Clyra-AI/axym/core/bundle"
	corecollect "github.com/Clyra-AI/axym/core/collect"
	"github.com/Clyra-AI/axym/core/collector"
	"github.com/Clyra-AI/axym/core/compliance/coverage"
	"github.com/Clyra-AI/axym/core/compliance/framework"
	"github.com/Clyra-AI/axym/core/compliance/match"
	"github.com/Clyra-AI/axym/core/gaps"
	"github.com/Clyra-AI/axym/core/policy/sink"
	"github.com/Clyra-AI/axym/core/store"
	verifybundle "github.com/Clyra-AI/axym/core/verify/bundle"
)

type runtimeBudgetFile struct {
	Workflows map[string]struct {
		P50MS int `json:"p50_ms"`
		P95MS int `json:"p95_ms"`
	} `json:"workflows"`
}

type benchBaselineFile struct {
	Workflows map[string]struct {
		MedianMS int `json:"median_ms"`
		P95MS    int `json:"p95_ms"`
	} `json:"workflows"`
}

type resourceBudgetFile struct {
	BundleMaxBytes     int64 `json:"bundle_max_bytes"`
	ManifestMaxEntries int   `json:"manifest_max_entries"`
	RawRecordMaxLines  int   `json:"raw_record_max_lines"`
	BundleFileMax      int   `json:"bundle_file_max"`
}

func requirePerfLane(t *testing.T) {
	t.Helper()
	if os.Getenv("AXYM_RUN_PERF") != "1" {
		t.Skip("perf tests run only in the perf lane")
	}
}

func TestPerfBenchBaselineMatchesRuntimeBudgetKeys(t *testing.T) {
	t.Parallel()
	requirePerfLane(t)

	repoRoot := repoRoot(t)
	var baselines benchBaselineFile
	loadJSONFile(t, filepath.Join(repoRoot, "perf", "bench_baseline.json"), &baselines)
	var budgets runtimeBudgetFile
	loadJSONFile(t, filepath.Join(repoRoot, "perf", "runtime_slo_budgets.json"), &budgets)

	if len(baselines.Workflows) == 0 || len(budgets.Workflows) == 0 {
		t.Fatal("expected non-empty benchmark and runtime budget fixtures")
	}
	for key := range budgets.Workflows {
		if _, ok := baselines.Workflows[key]; !ok {
			t.Fatalf("missing baseline entry for workflow %s", key)
		}
	}
}

func TestPerfCoreWorkflowBudgets(t *testing.T) {
	t.Parallel()
	requirePerfLane(t)

	prev := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(prev)

	repoRoot := repoRoot(t)
	var budgets runtimeBudgetFile
	loadJSONFile(t, filepath.Join(repoRoot, "perf", "runtime_slo_budgets.json"), &budgets)

	measurements := map[string][]time.Duration{
		"collect":       measureDurations(t, 5, func() { _, _ = runCollectWorkflow(t, repoRoot) }),
		"map_gaps":      measureDurations(t, 5, func() { runMapGapsWorkflow(t, repoRoot) }),
		"bundle_verify": measureDurations(t, 5, func() { runBundleVerifyWorkflow(t, repoRoot) }),
	}

	for workflow, samples := range measurements {
		budget, ok := budgets.Workflows[workflow]
		if !ok {
			t.Fatalf("missing runtime budget for workflow %s", workflow)
		}
		p50 := samples[len(samples)/2].Milliseconds()
		p95 := samples[len(samples)-1].Milliseconds()
		if p50 > int64(budget.P50MS) {
			t.Fatalf("%s p50 budget exceeded: got=%dms want<=%dms samples=%v", workflow, p50, budget.P50MS, samples)
		}
		if p95 > int64(budget.P95MS) {
			t.Fatalf("%s p95 budget exceeded: got=%dms want<=%dms samples=%v", workflow, p95, budget.P95MS, samples)
		}
	}
}

func TestPerfResourceBudgets(t *testing.T) {
	t.Parallel()
	requirePerfLane(t)

	repoRoot := repoRoot(t)
	var budgets resourceBudgetFile
	loadJSONFile(t, filepath.Join(repoRoot, "perf", "resource_budgets.json"), &budgets)

	storeDir, bundleDir := runBundleVerifyWorkflow(t, repoRoot)
	totalBytes, manifestEntries, bundleFiles, rawLines := bundleFootprint(t, bundleDir)
	if totalBytes > budgets.BundleMaxBytes {
		t.Fatalf("bundle size budget exceeded: got=%d want<=%d", totalBytes, budgets.BundleMaxBytes)
	}
	if manifestEntries > budgets.ManifestMaxEntries {
		t.Fatalf("manifest entry budget exceeded: got=%d want<=%d", manifestEntries, budgets.ManifestMaxEntries)
	}
	if bundleFiles > budgets.BundleFileMax {
		t.Fatalf("bundle file-count budget exceeded: got=%d want<=%d", bundleFiles, budgets.BundleFileMax)
	}
	if rawLines > budgets.RawRecordMaxLines {
		t.Fatalf("raw record line budget exceeded: got=%d want<=%d", rawLines, budgets.RawRecordMaxLines)
	}

	chain, err := os.ReadFile(filepath.Join(storeDir, "chain.json"))
	if err != nil {
		t.Fatalf("read chain: %v", err)
	}
	if len(chain) == 0 {
		t.Fatal("expected non-empty chain artifact")
	}
}

func TestPerfSoakWorkflowStability(t *testing.T) {
	t.Parallel()
	requirePerfLane(t)

	repoRoot := repoRoot(t)
	var last struct {
		CollectCaptured int
		GapCount        int
		BundleFiles     int
	}
	for i := 0; i < 5; i++ {
		storeDir, collectResult := runCollectWorkflow(t, repoRoot)
		gapCount := runMapGapsWorkflowWithStore(t, storeDir)
		bundleResult, err := corebundle.Build(corebundle.BuildRequest{
			AuditName:    "Q3-2026",
			FrameworkIDs: []string{"eu-ai-act", "soc2"},
			StoreDir:     storeDir,
			OutputDir:    filepath.Join(t.TempDir(), "bundle"),
		})
		if err != nil {
			t.Fatalf("bundle.Build: %v", err)
		}
		if _, err := verifybundle.Verify(bundleResult.Path, []string{"eu-ai-act", "soc2"}); err != nil {
			t.Fatalf("verifybundle.Verify: %v", err)
		}
		current := struct {
			CollectCaptured int
			GapCount        int
			BundleFiles     int
		}{
			CollectCaptured: collectResult.Captured,
			GapCount:        gapCount,
			BundleFiles:     bundleResult.Files,
		}
		if i > 0 && current != last {
			t.Fatalf("workflow stability mismatch: current=%+v last=%+v", current, last)
		}
		last = current
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

func loadJSONFile(t *testing.T, path string, target any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}

func measureDurations(t *testing.T, samples int, fn func()) []time.Duration {
	t.Helper()
	out := make([]time.Duration, 0, samples)
	for i := 0; i < samples; i++ {
		start := time.Now()
		fn()
		out = append(out, time.Since(start))
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i] < out[j]
	})
	return out
}

func runCollectWorkflow(t *testing.T, repoRoot string) (string, corecollect.Result) {
	t.Helper()

	storeDir := filepath.Join(t.TempDir(), "store")
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
	runner := corecollect.Runner{Registry: registry, Store: evidenceStore, SinkMode: sink.ModeFailClosed}
	result, err := runner.Run(context.Background(), req, false)
	if err != nil {
		t.Fatalf("runner.Run: %v", err)
	}
	return storeDir, result
}

func runMapGapsWorkflow(t *testing.T, repoRoot string) {
	t.Helper()
	storeDir, _ := runCollectWorkflow(t, repoRoot)
	_ = runMapGapsWorkflowWithStore(t, storeDir)
}

func runMapGapsWorkflowWithStore(t *testing.T, storeDir string) int {
	t.Helper()

	evidenceStore, err := store.New(store.Config{RootDir: storeDir, ComplianceMode: true})
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	chain, err := evidenceStore.LoadChain()
	if err != nil {
		t.Fatalf("LoadChain: %v", err)
	}
	definitions, err := framework.LoadMany([]string{"eu-ai-act", "soc2"})
	if err != nil {
		t.Fatalf("LoadMany: %v", err)
	}
	matchResult := match.Evaluate(definitions, chain.Records, match.Options{ExcludeInvalidEvidence: true})
	coverageReport := coverage.Build(matchResult)
	gapReport := gaps.Build(coverageReport)
	return gapReport.Summary.GapCount
}

func runBundleVerifyWorkflow(t *testing.T, repoRoot string) (string, string) {
	t.Helper()

	storeDir, _ := runCollectWorkflow(t, repoRoot)
	bundleDir := filepath.Join(t.TempDir(), "bundle")
	result, err := corebundle.Build(corebundle.BuildRequest{
		AuditName:    "Q3-2026",
		FrameworkIDs: []string{"eu-ai-act", "soc2"},
		StoreDir:     storeDir,
		OutputDir:    bundleDir,
	})
	if err != nil {
		t.Fatalf("bundle.Build: %v", err)
	}
	if _, err := verifybundle.Verify(result.Path, []string{"eu-ai-act", "soc2"}); err != nil {
		t.Fatalf("verifybundle.Verify: %v", err)
	}
	return storeDir, result.Path
}

func bundleFootprint(t *testing.T, bundleDir string) (int64, int, int, int) {
	t.Helper()

	var totalBytes int64
	fileCount := 0
	if err := filepath.Walk(bundleDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		fileCount++
		totalBytes += info.Size()
		return nil
	}); err != nil {
		t.Fatalf("walk bundle: %v", err)
	}

	rawManifest, err := os.ReadFile(filepath.Join(bundleDir, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest.json: %v", err)
	}
	var manifest struct {
		Files []struct {
			Path string `json:"path"`
		} `json:"files"`
	}
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		t.Fatalf("decode manifest.json: %v", err)
	}

	rawRecords, err := os.ReadFile(filepath.Join(bundleDir, "raw-records.jsonl"))
	if err != nil {
		t.Fatalf("read raw-records.jsonl: %v", err)
	}
	rawLines := 0
	for _, line := range strings.Split(strings.TrimSpace(string(rawRecords)), "\n") {
		if strings.TrimSpace(line) != "" {
			rawLines++
		}
	}
	return totalBytes, len(manifest.Files), fileCount, rawLines
}
