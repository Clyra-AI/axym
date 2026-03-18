package contracts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCIWorkflowFilesExist(t *testing.T) {
	t.Parallel()

	required := []string{
		".github/workflows/pr.yml",
		".github/workflows/main.yml",
		".github/workflows/nightly.yml",
		".github/workflows/release.yml",
		".goreleaser.yaml",
		"docs/commands/axym.md",
		"perf/bench_baseline.json",
		"perf/runtime_slo_budgets.json",
		"perf/resource_budgets.json",
		"scripts/check_branch_protection_contract.sh",
		"scripts/release_go_nogo.sh",
		"scripts/validate_scenarios.sh",
	}

	for _, path := range required {
		if _, err := os.Stat(filepath.Join(testRepoRoot(t), filepath.FromSlash(path))); err != nil {
			t.Fatalf("required CI artifact missing: %s: %v", path, err)
		}
	}
}

func TestWorkflowsDoNotUseLatestFloatingTags(t *testing.T) {
	t.Parallel()

	files := []string{
		".github/workflows/pr.yml",
		".github/workflows/main.yml",
		".github/workflows/nightly.yml",
		".github/workflows/release.yml",
	}
	for _, path := range files {
		content := readRepoFile(t, path)
		if strings.Contains(content, "@latest") {
			t.Fatalf("workflow has floating @latest reference: %s", path)
		}
	}
}
