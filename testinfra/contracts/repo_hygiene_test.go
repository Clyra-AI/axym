package contracts

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestRequiredPlanningArtifactsExist(t *testing.T) {
	t.Parallel()

	required := []string{
		"CONTRIBUTING.md",
		"SECURITY.md",
		"product/PLAN_v1.0.md",
	}

	for _, path := range required {
		if _, err := os.Stat(filepath.Join(repoRoot(t), filepath.FromSlash(path))); err != nil {
			t.Fatalf("required planning artifact missing: %s: %v", path, err)
		}
	}
}

func TestRequiredLaunchAssetsExist(t *testing.T) {
	t.Parallel()

	required := []string{
		"LICENSE",
		"CHANGELOG.md",
		"CODE_OF_CONDUCT.md",
		".github/ISSUE_TEMPLATE/bug_report.yml",
		".github/ISSUE_TEMPLATE/feature_request.yml",
		".github/pull_request_template.md",
	}

	for _, path := range required {
		if _, err := os.Stat(filepath.Join(repoRoot(t), filepath.FromSlash(path))); err != nil {
			t.Fatalf("required launch asset missing: %s: %v", path, err)
		}
	}
}

func TestLaunchFacingDocsReferenceCurrentOSSBoundary(t *testing.T) {
	t.Parallel()

	for _, doc := range loadLaunchNarrativeDocs(t) {
		for _, snippet := range []string{
			"Smoke test",
			"Sample proof path",
			"Real integration path",
			"First value is evidence + ranked gaps + intact local verification, not full audit completeness.",
			"./axym init --sample-pack ./axym-sample --json",
		} {
			if !strings.Contains(doc.content, snippet) {
				t.Fatalf("launch-facing docs missing snippet %q in %s", snippet, doc.path)
			}
		}
	}
}

func TestRepoHygieneContributorDocsListFullGatePrerequisites(t *testing.T) {
	t.Parallel()

	requiredCommands := []string{
		"make prepush-full",
		"make release-local",
		"make release-go-nogo-local",
		"./scripts/release_go_nogo.sh --dist-dir dist --binary-name axym",
	}
	requiredTools := []string{
		"golangci-lint",
		"gosec",
		"codeql",
		"syft",
		"cosign",
	}

	for _, path := range []string{"README.md", "CONTRIBUTING.md", "docs-site/public/llms.txt"} {
		content := readRepoFile(t, path)
		for _, command := range requiredCommands {
			if !strings.Contains(content, command) {
				t.Fatalf("%s missing prerequisite command %q", path, command)
			}
		}
		for _, tool := range requiredTools {
			if !strings.Contains(content, tool) {
				t.Fatalf("%s missing prerequisite tool %q", path, tool)
			}
		}
	}
}

func TestRepoHygieneSecurityReportingPathsStayExplicit(t *testing.T) {
	t.Parallel()

	security := readRepoFile(t, "SECURITY.md")
	for _, snippet := range []string{
		"GitHub Security Advisories",
		"minimal public GitHub issue without exploit details",
	} {
		if !strings.Contains(security, snippet) {
			t.Fatalf("SECURITY.md missing security reporting snippet %q", snippet)
		}
	}

	for _, path := range []string{"README.md", "CONTRIBUTING.md", "docs-site/public/llms.txt", ".github/ISSUE_TEMPLATE/bug_report.yml"} {
		content := readRepoFile(t, path)
		for _, snippet := range []string{
			"SECURITY.md",
			"GitHub Security Advisories",
		} {
			if !strings.Contains(content, snippet) {
				t.Fatalf("%s missing security routing snippet %q", path, snippet)
			}
		}
	}
}

func TestTrackedSecretArtifactsAreAbsent(t *testing.T) {
	t.Parallel()

	prohibited := []string{
		".env",
		"id_rsa",
		"secrets.txt",
	}

	for _, path := range prohibited {
		if _, err := os.Stat(filepath.Join(repoRoot(t), filepath.FromSlash(path))); err == nil {
			t.Fatalf("prohibited artifact exists: %s", path)
		}
	}
}
