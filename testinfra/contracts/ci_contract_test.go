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
		".github/workflows/codeql.yml",
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
		"scripts/check_docs_links.sh",
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
		".github/workflows/codeql.yml",
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

func TestWorkflowsUseNode24ReadyPinnedActions(t *testing.T) {
	t.Parallel()

	required := map[string][]string{
		".github/workflows/codeql.yml": {
			"actions/checkout@v6",
			"actions/setup-go@v6",
			"github/codeql-action/init@v4",
			"github/codeql-action/analyze@v4",
		},
		".github/workflows/pr.yml": {
			"actions/checkout@v6",
			"actions/setup-go@v6",
		},
		".github/workflows/main.yml": {
			"actions/checkout@v6",
			"actions/setup-go@v6",
		},
		".github/workflows/nightly.yml": {
			"actions/checkout@v6",
			"actions/setup-go@v6",
		},
		".github/workflows/release.yml": {
			"actions/checkout@v6",
			"actions/setup-go@v6",
			"goreleaser/goreleaser-action@v7",
			"sigstore/cosign-installer@v4.1.0",
			"actions/attest-build-provenance@v4",
		},
	}

	for path, snippets := range required {
		content := readRepoFile(t, path)
		for _, snippet := range snippets {
			if !strings.Contains(content, snippet) {
				t.Fatalf("workflow missing Node24-ready pinned action %q in %s", snippet, path)
			}
		}
	}
}

func TestCodeQLWorkflowTriggersOnPullRequestAndMain(t *testing.T) {
	t.Parallel()

	content := readRepoFile(t, ".github/workflows/codeql.yml")
	required := []string{
		"pull_request:",
		"push:",
		"- main",
		"workflow_dispatch:",
		"security-events: write",
	}
	for _, snippet := range required {
		if !strings.Contains(content, snippet) {
			t.Fatalf("codeql workflow missing snippet %q", snippet)
		}
	}
	if strings.Contains(content, "upload: never") {
		t.Fatal("codeql workflow must upload hosted analysis results on pull_request and main")
	}
}

func TestWorkflowsDoNotOptIntoInsecureNodeFallback(t *testing.T) {
	t.Parallel()

	files := []string{
		".github/workflows/pr.yml",
		".github/workflows/main.yml",
		".github/workflows/nightly.yml",
		".github/workflows/release.yml",
	}
	for _, path := range files {
		content := readRepoFile(t, path)
		if strings.Contains(content, "ACTIONS_ALLOW_USE_UNSECURE_NODE_VERSION") {
			t.Fatalf("workflow opts into insecure node fallback: %s", path)
		}
	}
}
